package db

import (
	"database/sql"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var dbPath = filepath.Join("..", "data", "db.sqlite")

func getDB() (*sql.DB, error) {
	return sql.Open("sqlite3", dbPath)
}
