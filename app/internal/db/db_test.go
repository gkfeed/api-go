package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"gkfeed/api/internal/passwordhash"
)

func TestMigratePasswords(t *testing.T) {
	database := useTestDatabase(t)
	if _, err := database.Exec("INSERT INTO users (id, name, hashed_password) VALUES (1, 'reader', 'secret')"); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}
	user, err := GetUserFromDB("reader")
	if err != nil {
		t.Fatalf("GetUserFromDB() returned error: %v", err)
	}
	if !passwordhash.ComparePassword(user.HashedPassword, "secret") {
		t.Fatal("MigratePasswords() did not store a matching hash")
	}
	hash := user.HashedPassword
	if err := RunMigrations(); err != nil {
		t.Fatalf("second RunMigrations() returned error: %v", err)
	}
	user, _ = GetUserFromDB("reader")
	if user.HashedPassword != hash {
		t.Fatal("MigratePasswords() rehashed an encoded password")
	}
}

func TestMigratePasswordsLeavesNullPasswordsAlone(t *testing.T) {
	database := useTestDatabase(t)
	if _, err := database.Exec("INSERT INTO users (id, name, hashed_password) VALUES (2, 'no-password', NULL)"); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}
	var password sql.NullString
	if err := database.QueryRow("SELECT hashed_password FROM users WHERE id = 2").Scan(&password); err != nil {
		t.Fatal(err)
	}
	if password.Valid {
		t.Fatalf("password became %q", password.String)
	}
}

func useTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite3", t.TempDir()+"/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	Configure(database)
	t.Cleanup(func() {
		Configure(nil)
		database.Close()
	})
	if err := RunMigrations(); err != nil {
		t.Fatal(err)
	}
	return database
}
