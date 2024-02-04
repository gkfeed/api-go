package db

import "database/sql"

func getDB() (*sql.DB, error) {
	return sql.Open("sqlite3", "../data/db.sqlite")
}
