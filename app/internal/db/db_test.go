package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"gkfeed/api/internal/models"
	"gkfeed/api/internal/passwordhash"
)

func TestFeedAndItemQueries(t *testing.T) {
	useTestDatabase(t)

	feed, err := AddFeed(models.Feed{
		Title: "Example",
		Type:  "yt",
		URL:   "https://www.youtube.com/@example",
	}, 1)
	if err != nil {
		t.Fatalf("AddFeed() returned an error: %v", err)
	}
	if feed.ID == 0 {
		t.Fatal("AddFeed() returned a feed without an ID")
	}

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	_, err = database.Exec(
		"INSERT INTO item (feed_id, title, text, date, link) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)",
		feed.ID, "First", "first", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "https://example.com/first",
		feed.ID, "Second", "second", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), "https://example.com/second",
	)
	database.Close()
	if err != nil {
		t.Fatalf("insert test items: %v", err)
	}

	items, err := GetUserItemsPage(1, nil, 1)
	if err != nil {
		t.Fatalf("GetUserItemsPage() returned an error: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Second" {
		t.Fatalf("GetUserItemsPage() = %#v, want the item with the highest ID", items)
	}

	cursor := items[0].ID
	remaining, err := GetUserItemsPage(1, &cursor, 10)
	if err != nil {
		t.Fatalf("GetUserItemsPage() after cursor returned an error: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("GetUserItemsPage() after cursor returned %d items, want 1", len(remaining))
	}

	if err := InsertItemsIntoDeletedItems(1, []int{items[0].ID}); err != nil {
		t.Fatalf("InsertItemsIntoDeletedItems() returned an error: %v", err)
	}
	visible, err := GetUserItems(1)
	if err != nil {
		t.Fatalf("GetUserItems() returned an error: %v", err)
	}
	if len(visible) != 1 {
		t.Fatalf("GetUserItems() returned %d visible items after deletion, want 1", len(visible))
	}
}

func TestMigratePasswords(t *testing.T) {
	useTestDatabase(t)

	if err := RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}

	user, err := GetUserFromDB("reader")
	if err != nil {
		t.Fatalf("GetUserFromDB() returned error: %v", err)
	}
	if user.HashedPassword == "secret" {
		t.Fatal("MigratePasswords() left the plaintext password in the database")
	}
	if !passwordhash.ComparePassword(user.HashedPassword, "secret") {
		t.Fatal("MigratePasswords() stored a hash that does not match the password")
	}

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	var hashBeforeSecondMigration string
	if err := database.QueryRow("SELECT password FROM users WHERE id = 1").Scan(&hashBeforeSecondMigration); err != nil {
		database.Close()
		t.Fatalf("read migrated password: %v", err)
	}
	database.Close()

	if err := RunMigrations(); err != nil {
		t.Fatalf("second RunMigrations() returned error: %v", err)
	}

	database, err = getDB()
	if err != nil {
		t.Fatalf("reopen test database: %v", err)
	}
	defer database.Close()
	var hashAfterSecondMigration string
	if err := database.QueryRow("SELECT password FROM users WHERE id = 1").Scan(&hashAfterSecondMigration); err != nil {
		t.Fatalf("read password after second migration: %v", err)
	}
	if hashAfterSecondMigration != hashBeforeSecondMigration {
		t.Fatal("MigratePasswords() rehashed an already migrated password")
	}
}

func TestMigratePasswordsLeavesNullPasswordsAlone(t *testing.T) {
	useTestDatabase(t)

	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	_, err = database.Exec("INSERT INTO users (id, name, password) VALUES (?, ?, NULL)", 2, "no-password")
	database.Close()
	if err != nil {
		t.Fatalf("insert null password: %v", err)
	}

	if err := RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}

	database, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("reopen test database: %v", err)
	}
	defer database.Close()
	var password sql.NullString
	if err := database.QueryRow("SELECT password FROM users WHERE id = 2").Scan(&password); err != nil {
		t.Fatalf("read null password: %v", err)
	}
	if password.Valid {
		t.Fatalf("MigratePasswords() changed a NULL password to %q", password.String)
	}
}

func useTestDatabase(t *testing.T) {
	t.Helper()

	originalPath := dbPath
	dbPath = filepath.Join(t.TempDir(), "db.sqlite")
	t.Cleanup(func() { dbPath = originalPath })

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	schema := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, password TEXT)",
		"CREATE TABLE feed (id INTEGER PRIMARY KEY, title TEXT, url TEXT, type TEXT, user_id INTEGER)",
		"CREATE TABLE item (id INTEGER PRIMARY KEY, feed_id INTEGER, title TEXT, text TEXT, date DATETIME, link TEXT)",
		"CREATE TABLE deleted_items (user_id INTEGER, item_id INTEGER)",
		"CREATE TABLE webauthn_credentials (id BLOB PRIMARY KEY, user_id INTEGER NOT NULL, credential TEXT NOT NULL, name TEXT NOT NULL DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, last_used_at DATETIME)",
		"CREATE TABLE refresh_tokens (id TEXT PRIMARY KEY, user_id INTEGER NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
		"INSERT INTO users (id, name, password) VALUES (1, 'reader', 'secret')",
	}
	for _, statement := range schema {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("initialize test database: %v", err)
		}
	}
}
