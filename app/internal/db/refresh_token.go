package db

import (
	"database/sql"
	"errors"
	"fmt"

	"gkfeed/api/internal/models"
)

func StoreRefreshToken(token models.RefreshToken) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	_, err = database.Exec(
		"INSERT INTO refresh_tokens (id, user_id, expires_at) VALUES ($1, $2, $3)",
		token.ID, token.UserID, token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func GetRefreshToken(id string) (models.RefreshToken, error) {
	database, err := getDB()
	if err != nil {
		return models.RefreshToken{}, fmt.Errorf("open database: %w", err)
	}
	var token models.RefreshToken
	err = database.QueryRow(
		"SELECT id, user_id, expires_at, created_at FROM refresh_tokens WHERE id = $1",
		id,
	).Scan(&token.ID, &token.UserID, &token.ExpiresAt, &token.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.RefreshToken{}, fmt.Errorf("refresh token not found: %w", err)
	}
	if err != nil {
		return models.RefreshToken{}, fmt.Errorf("query refresh token: %w", err)
	}
	return token, nil
}

func DeleteRefreshToken(id string) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	_, err = database.Exec("DELETE FROM refresh_tokens WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}

func DeleteUserRefreshTokens(userID int) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	_, err = database.Exec("DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete user refresh tokens: %w", err)
	}
	return nil
}
