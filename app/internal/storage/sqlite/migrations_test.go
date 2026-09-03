package sqlite

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrateDeletedItemsNoTableIsNoOp(t *testing.T) {
	database := newTestDB(t)
	stats, err := MigrateDeletedItems(t.Context(), database)
	if err != nil || stats != (DeletedItemsMigrationStats{}) {
		t.Fatalf("MigrateDeletedItems() = %#v, %v", stats, err)
	}
}

func TestMigrateDeletedItemsDeletesOnlyValidOwnedItems(t *testing.T) {
	database := newTestDB(t)
	mustExec(t, database, "CREATE TABLE deleted_items (user_id INTEGER, item_id INTEGER)")
	mustExec(t, database, "INSERT INTO feed (id, user_id) VALUES (1, 10), (2, 20)")
	mustExec(t, database, "INSERT INTO item (id, feed_id) VALUES (101, 1), (102, 1), (201, 2)")
	// Two duplicate valid rows, plus foreign, missing-item and missing-user rows.
	mustExec(t, database, "INSERT INTO deleted_items VALUES (10, 101), (10, 101), (10, 201), (10, 999), (999, 102)")

	stats, err := MigrateDeletedItems(t.Context(), database)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Deleted != 1 || stats.Ignored != 3 {
		t.Fatalf("stats = %#v", stats)
	}
	for _, test := range []struct {
		id     int
		exists bool
	}{{101, false}, {102, true}, {201, true}} {
		if got := rowExists(t, database, "item", test.id); got != test.exists {
			t.Fatalf("item %d exists = %t", test.id, got)
		}
	}
	var tables int
	if err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='deleted_items'").Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("deleted_items table count = %d, err = %v", tables, err)
	}
	stats, err = MigrateDeletedItems(t.Context(), database)
	if err != nil || stats != (DeletedItemsMigrationStats{}) {
		t.Fatalf("second migration = %#v, %v", stats, err)
	}
}

func TestMigrateDeletedItemsRollsBack(t *testing.T) {
	database := newTestDB(t)
	mustExec(t, database, "CREATE TABLE deleted_items (user_id INTEGER, item_id INTEGER)")
	mustExec(t, database, "INSERT INTO feed (id, user_id) VALUES (1, 10)")
	mustExec(t, database, "INSERT INTO item (id, feed_id) VALUES (101, 1)")
	mustExec(t, database, "INSERT INTO deleted_items VALUES (10, 101)")
	mustExec(t, database, `CREATE TRIGGER reject_item_delete BEFORE DELETE ON item BEGIN SELECT RAISE(ABORT, 'rejected'); END`)
	if _, err := MigrateDeletedItems(t.Context(), database); err == nil {
		t.Fatal("migration succeeded despite trigger")
	}
	if !rowExists(t, database, "item", 101) {
		t.Fatal("item was deleted despite rollback")
	}
	var tables int
	if err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='deleted_items'").Scan(&tables); err != nil || tables != 1 {
		t.Fatalf("deleted_items table count = %d, err = %v", tables, err)
	}
}

func mustExec(t *testing.T, database *sql.DB, statement string) {
	t.Helper()
	if _, err := database.Exec(statement); err != nil {
		t.Fatal(err)
	}
}

func rowExists(t *testing.T, database *sql.DB, table string, id int) bool {
	t.Helper()
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id = ?", id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count == 1
}
