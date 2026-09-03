package db

import (
	"database/sql"
	"errors"
	"sync"
)

var (
	databaseMu sync.RWMutex
	database   *sql.DB
)

// Configure temporarily wires legacy auth persistence to the shared pool.
// TODO: move auth repositories to constructor injection.
func Configure(db *sql.DB) {
	databaseMu.Lock()
	database = db
	databaseMu.Unlock()
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
	schema := []string{
		"CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, hashed_password TEXT)",
	}
	for _, statement := range schema {
		if _, err := database.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func getDB() (*sql.DB, error) {
	databaseMu.RLock()
	defer databaseMu.RUnlock()
	if database == nil {
		return nil, errors.New("database is not configured")
	}
	return database, nil
}

// ConfiguredDB exists only while legacy auth storage still uses package-level wiring.
func ConfiguredDB() (*sql.DB, error) { return getDB() }
