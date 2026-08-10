package db

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

var dbPath string

func Configure(path string) {
	dbPath = path
}

func RunMigrations() error {
	if err := InitCoreSchema(); err != nil {
		return err
	}
	if err := MigratePasswords(); err != nil {
		return err
	}
	if err := InitWebAuthnSchema(); err != nil {
		return err
	}
	if err := InitRefreshTokenSchema(); err != nil {
		return err
	}
	return nil
}

func InitCoreSchema() error {
	database, err := getDB()
	if err != nil {
		return err
	}
	defer database.Close()

	schema := []string{
		"CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, hashed_password TEXT)",
		"CREATE TABLE IF NOT EXISTS feed (id INTEGER PRIMARY KEY, title TEXT, url TEXT, type TEXT, user_id INTEGER)",
		"CREATE TABLE IF NOT EXISTS item (id INTEGER PRIMARY KEY, feed_id INTEGER, title TEXT, text TEXT, date DATETIME, link TEXT)",
		"CREATE TABLE IF NOT EXISTS deleted_items (user_id INTEGER, item_id INTEGER)",
		`CREATE TABLE IF NOT EXISTS inbox_deliveries (
			id TEXT PRIMARY KEY,
			sender_user_id INTEGER NOT NULL,
			recipient_user_id INTEGER NOT NULL,
			cloned_item_id INTEGER NOT NULL UNIQUE,
			note TEXT,
			idempotency_key TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(sender_user_id, idempotency_key)
		)`,
		"CREATE UNIQUE INDEX IF NOT EXISTS one_inbox_per_user ON feed(user_id) WHERE type = 'inbox'",
		"CREATE INDEX IF NOT EXISTS inbox_deliveries_recipient ON inbox_deliveries(recipient_user_id, created_at DESC)",
	}
	for _, statement := range schema {
		if _, err := database.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func getDB() (*sql.DB, error) {
	if dbPath == "" {
		return nil, errors.New("database is not configured")
	}
	return sql.Open("sqlite3", dbPath)
}
