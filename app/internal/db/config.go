package db

import (
	"database/sql"
	"path/filepath"
)

var dbPath = filepath.Join("data", "db.sqlite")

func getDB() (*sql.DB, error) {
	return sql.Open("sqlite3", dbPath)
}
