package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

type DeletedItemsMigrationStats struct {
	Deleted int64
	Ignored int64
}

func InitSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range []string{
		"CREATE TABLE IF NOT EXISTS feed (id INTEGER PRIMARY KEY, title TEXT, url TEXT, type TEXT, user_id INTEGER)",
		"CREATE TABLE IF NOT EXISTS item (id INTEGER PRIMARY KEY, feed_id INTEGER, title TEXT, text TEXT, date DATETIME, link TEXT)",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize library schema: %w", err)
		}
	}
	return nil
}

func MigrateDeletedItems(ctx context.Context, db *sql.DB) (DeletedItemsMigrationStats, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("begin deleted-items migration: %w", err)
	}
	defer tx.Rollback()

	var tableCount int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'deleted_items'`).Scan(&tableCount); err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("check deleted_items table: %w", err)
	}
	if tableCount == 0 {
		if err := tx.Commit(); err != nil {
			return DeletedItemsMigrationStats{}, fmt.Errorf("commit deleted-items no-op: %w", err)
		}
		return DeletedItemsMigrationStats{}, nil
	}

	var total int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM deleted_items").Scan(&total); err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("count deleted-item tombstones: %w", err)
	}
	var validRows int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM deleted_items d
		JOIN item i ON i.id = d.item_id
		JOIN feed f ON f.id = i.feed_id AND f.user_id = d.user_id`).Scan(&validRows); err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("count valid tombstones: %w", err)
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM item WHERE id IN (
		SELECT DISTINCT i.id FROM deleted_items d
		JOIN item i ON i.id = d.item_id
		JOIN feed f ON f.id = i.feed_id AND f.user_id = d.user_id
	)`)
	if err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("delete tombstoned items: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("count deleted items: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DROP TABLE deleted_items"); err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("drop deleted_items table: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return DeletedItemsMigrationStats{}, fmt.Errorf("commit deleted-items migration: %w", err)
	}
	return DeletedItemsMigrationStats{Deleted: deleted, Ignored: total - validRows}, nil
}
