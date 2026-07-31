package db

import (
	"fmt"

	"gkfeed/api/internal/passwordhash"
)

// MigratePasswords replaces legacy plaintext passwords with Argon2id hashes.
// It is safe to run repeatedly: already encoded passwords are left unchanged.
func MigratePasswords() error {
	database, err := getDB()
	if err != nil {
		return err
	}
	defer database.Close()

	transaction, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin password migration: %w", err)
	}
	defer transaction.Rollback()

	rows, err := transaction.Query("SELECT id, password FROM users WHERE password IS NOT NULL")
	if err != nil {
		return fmt.Errorf("read passwords: %w", err)
	}

	type legacyPassword struct {
		id       int
		password string
	}
	var passwords []legacyPassword
	for rows.Next() {
		var password legacyPassword
		if err := rows.Scan(&password.id, &password.password); err != nil {
			rows.Close()
			return fmt.Errorf("read password: %w", err)
		}
		if !passwordhash.IsEncoded(password.password) {
			passwords = append(passwords, password)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate passwords: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close password rows: %w", err)
	}

	for _, legacy := range passwords {
		hashed, err := passwordhash.HashPassword(legacy.password)
		if err != nil {
			return fmt.Errorf("hash password for user %d: %w", legacy.id, err)
		}
		if _, err := transaction.Exec(
			"UPDATE users SET password = ? WHERE id = ?",
			hashed,
			legacy.id,
		); err != nil {
			return fmt.Errorf("update password for user %d: %w", legacy.id, err)
		}
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit password migration: %w", err)
	}
	return nil
}
