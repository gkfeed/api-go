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

func getDB() (*sql.DB, error) {
	if dbPath == "" {
		return nil, errors.New("database is not configured")
	}
	return sql.Open("sqlite3", dbPath)
}
