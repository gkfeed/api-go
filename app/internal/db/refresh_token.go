package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gkfeed/api/internal/models"
)

var ErrInvalidRefreshToken = errors.New("invalid refresh token")

var ErrRefreshTokenReuse = errors.New("refresh token reuse detected")

// RefreshRotation contains the user and family that belong to a refresh
// token. It is returned alongside an error when a token is expired or reused,
// so callers can revoke the corresponding access-token family as well.
type RefreshRotation struct {
	User     models.User
	FamilyID []byte
}

type refreshTokenReuseError struct {
	RefreshRotation
}

func (e *refreshTokenReuseError) Error() string {
	return ErrRefreshTokenReuse.Error()
}

func (e *refreshTokenReuseError) Unwrap() error {
	return ErrRefreshTokenReuse
}

func InitRefreshTokenSchema() error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	_, err = database.Exec(`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create refresh_tokens table: %w", err)
	}

	_, err = database.Exec(`CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token_hash BLOB NOT NULL UNIQUE,
		family_id BLOB NOT NULL,
		user_id INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		revoked_at INTEGER,
		replaced_by BLOB
	);
	CREATE INDEX IF NOT EXISTS auth_refresh_tokens_family
		ON auth_refresh_tokens (family_id);
	CREATE INDEX IF NOT EXISTS auth_refresh_tokens_user
		ON auth_refresh_tokens (user_id);`)
	if err != nil {
		return fmt.Errorf("create auth_refresh_tokens table: %w", err)
	}
	return nil
}

func CreateAuthRefreshToken(userID int, tokenHash, familyID []byte, expiresAt time.Time) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	_, err = database.Exec(
		`INSERT INTO auth_refresh_tokens
			(token_hash, family_id, user_id, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		tokenHash,
		familyID,
		userID,
		expiresAt.Unix(),
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert auth refresh token: %w", err)
	}
	return nil
}

// RotateAuthRefreshToken atomically consumes oldHash and creates the next
// token in the same family. SQLite's immediate transaction lock makes two
// simultaneous refresh requests deterministic: only one can consume a token.
func RotateAuthRefreshToken(oldHash, newHash []byte, now, expiresAt time.Time) (RefreshRotation, error) {
	if dbPath == "" {
		return RefreshRotation{}, errors.New("database is not configured")
	}

	database, err := sql.Open("sqlite3", dbPath+"?_txlock=immediate&_busy_timeout=5000")
	if err != nil {
		return RefreshRotation{}, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	transaction, err := database.Begin()
	if err != nil {
		return RefreshRotation{}, fmt.Errorf("begin refresh-token transaction: %w", err)
	}
	defer transaction.Rollback()

	var (
		familyID  []byte
		userID    int
		userName  string
		expires   int64
		revokedAt sql.NullInt64
		replaced  []byte
	)
	err = transaction.QueryRow(
		`SELECT token.family_id, token.user_id, users.name, token.expires_at,
				token.revoked_at, token.replaced_by
		   FROM auth_refresh_tokens AS token
		   JOIN users ON users.id = token.user_id
		  WHERE token.token_hash = ?`,
		oldHash,
	).Scan(&familyID, &userID, &userName, &expires, &revokedAt, &replaced)
	if errors.Is(err, sql.ErrNoRows) {
		return RefreshRotation{}, fmt.Errorf("%w: token not found", ErrInvalidRefreshToken)
	}
	if err != nil {
		return RefreshRotation{}, fmt.Errorf("query auth refresh token: %w", err)
	}

	rotation := RefreshRotation{
		User:     models.User{ID: userID, Name: userName},
		FamilyID: append([]byte(nil), familyID...),
	}
	if revokedAt.Valid || len(replaced) > 0 {
		if err := revokeAuthRefreshFamily(transaction, rotation.FamilyID, now.Unix()); err != nil {
			return RefreshRotation{}, err
		}
		if err := transaction.Commit(); err != nil {
			return RefreshRotation{}, fmt.Errorf("commit refresh-token reuse revocation: %w", err)
		}
		return rotation, &refreshTokenReuseError{RefreshRotation: rotation}
	}
	if expires <= now.Unix() {
		if err := revokeAuthRefreshFamily(transaction, rotation.FamilyID, now.Unix()); err != nil {
			return RefreshRotation{}, err
		}
		if err := transaction.Commit(); err != nil {
			return RefreshRotation{}, fmt.Errorf("commit expired refresh-token revocation: %w", err)
		}
		return rotation, fmt.Errorf("%w: token expired", ErrInvalidRefreshToken)
	}

	result, err := transaction.Exec(
		`UPDATE auth_refresh_tokens
			SET revoked_at = ?, replaced_by = ?
		  WHERE token_hash = ? AND revoked_at IS NULL AND replaced_by IS NULL`,
		now.Unix(),
		newHash,
		oldHash,
	)
	if err != nil {
		return RefreshRotation{}, fmt.Errorf("consume auth refresh token: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return RefreshRotation{}, fmt.Errorf("auth refresh token rows affected: %w", err)
	}
	if rows != 1 {
		if err := revokeAuthRefreshFamily(transaction, rotation.FamilyID, now.Unix()); err != nil {
			return RefreshRotation{}, err
		}
		if err := transaction.Commit(); err != nil {
			return RefreshRotation{}, fmt.Errorf("commit refresh-token reuse revocation: %w", err)
		}
		return rotation, &refreshTokenReuseError{RefreshRotation: rotation}
	}

	_, err = transaction.Exec(
		`INSERT INTO auth_refresh_tokens
			(token_hash, family_id, user_id, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		newHash,
		rotation.FamilyID,
		rotation.User.ID,
		expiresAt.Unix(),
		now.Unix(),
	)
	if err != nil {
		return RefreshRotation{}, fmt.Errorf("insert rotated refresh token: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return RefreshRotation{}, fmt.Errorf("commit refresh-token rotation: %w", err)
	}
	return rotation, nil
}

func RevokeAuthRefreshToken(tokenHash []byte, now time.Time) ([]byte, error) {
	database, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	var familyID []byte
	if err := database.QueryRow(
		"SELECT family_id FROM auth_refresh_tokens WHERE token_hash = ?",
		tokenHash,
	).Scan(&familyID); errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: token not found", ErrInvalidRefreshToken)
	} else if err != nil {
		return nil, fmt.Errorf("query auth refresh token family: %w", err)
	}

	if _, err := database.Exec(
		`UPDATE auth_refresh_tokens
			SET revoked_at = COALESCE(revoked_at, ?)
		  WHERE family_id = ?`,
		now.Unix(),
		familyID,
	); err != nil {
		return nil, fmt.Errorf("revoke auth refresh-token family: %w", err)
	}
	return familyID, nil
}

func RevokeUserAuthRefreshTokens(userID int, now time.Time) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	if _, err := database.Exec(
		`UPDATE auth_refresh_tokens
			SET revoked_at = COALESCE(revoked_at, ?)
		  WHERE user_id = ?`,
		now.Unix(),
		userID,
	); err != nil {
		return fmt.Errorf("revoke user auth refresh tokens: %w", err)
	}
	return nil
}

func revokeAuthRefreshFamily(transaction *sql.Tx, familyID []byte, now int64) error {
	if _, err := transaction.Exec(
		`UPDATE auth_refresh_tokens
			SET revoked_at = COALESCE(revoked_at, ?)
		  WHERE family_id = ?`,
		now,
		familyID,
	); err != nil {
		return fmt.Errorf("revoke auth refresh-token family: %w", err)
	}
	return nil
}

func StoreRefreshToken(token models.RefreshToken) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	_, err = database.Exec(
		"INSERT INTO refresh_tokens (id, user_id, expires_at) VALUES (?, ?, ?)",
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
	defer database.Close()

	var token models.RefreshToken
	err = database.QueryRow(
		"SELECT id, user_id, expires_at, created_at FROM refresh_tokens WHERE id = ?",
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
	defer database.Close()

	_, err = database.Exec("DELETE FROM refresh_tokens WHERE id = ?", id)
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
	defer database.Close()

	_, err = database.Exec("DELETE FROM refresh_tokens WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("delete user refresh tokens: %w", err)
	}
	return nil
}
