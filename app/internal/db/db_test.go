package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"gkfeed/api/internal/models"
	"gkfeed/api/internal/passwordhash"
	"gkfeed/api/internal/testdb"

	"github.com/go-webauthn/webauthn/webauthn"
)

func TestFeedAndItemQueries(t *testing.T) {
	useTestDatabase(t)

	feed, err := AddFeed(models.Feed{
		Title: "Example",
		Type:  "yt",
		URL:   "https://www.youtube.com/@example",
	}, 1)
	if err != nil {
		t.Fatalf("AddFeed() returned an error: %v", err)
	}
	if feed.ID == 0 {
		t.Fatal("AddFeed() returned a feed without an ID")
	}

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	_, err = database.Exec(
		"INSERT INTO item (feed_id, title, text, date, link) VALUES ($1, $2, $3, $4, $5), ($6, $7, $8, $9, $10)",
		feed.ID, "First", "first", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "https://example.com/first",
		feed.ID, "Second", "second", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), "https://example.com/second",
	)
	if err != nil {
		t.Fatalf("insert test items: %v", err)
	}

	items, err := GetUserItemsPage(1, nil, 1)
	if err != nil {
		t.Fatalf("GetUserItemsPage() returned an error: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Second" {
		t.Fatalf("GetUserItemsPage() = %#v, want the item with the highest ID", items)
	}

	cursor := items[0].ID
	remaining, err := GetUserItemsPage(1, &cursor, 10)
	if err != nil {
		t.Fatalf("GetUserItemsPage() after cursor returned an error: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("GetUserItemsPage() after cursor returned %d items, want 1", len(remaining))
	}

	if err := InsertItemsIntoDeletedItems(1, []int{items[0].ID}); err != nil {
		t.Fatalf("InsertItemsIntoDeletedItems() returned an error: %v", err)
	}
	visible, err := GetUserItems(1)
	if err != nil {
		t.Fatalf("GetUserItems() returned an error: %v", err)
	}
	if len(visible) != 1 {
		t.Fatalf("GetUserItems() returned %d visible items after deletion, want 1", len(visible))
	}

	item, itemFeed, err := GetUserItemByID(1, remaining[0].ID)
	if err != nil {
		t.Fatalf("GetUserItemByID() returned an error: %v", err)
	}
	if item.ID != remaining[0].ID || itemFeed.ID != feed.ID || itemFeed.UserID != 1 {
		t.Fatalf("GetUserItemByID() = (%#v, %#v), want item and feed owned by user 1", item, itemFeed)
	}

	if _, _, err := GetUserItemByID(2, remaining[0].ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetUserItemByID() for another user returned error %v, want sql.ErrNoRows", err)
	}
}

func TestMigratePasswords(t *testing.T) {
	useTestDatabase(t)

	if err := RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}

	user, err := GetUserFromDB("reader")
	if err != nil {
		t.Fatalf("GetUserFromDB() returned error: %v", err)
	}
	if user.HashedPassword == "secret" {
		t.Fatal("MigratePasswords() left the plaintext password in the database")
	}
	if !passwordhash.ComparePassword(user.HashedPassword, "secret") {
		t.Fatal("MigratePasswords() stored a hash that does not match the password")
	}

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	var hashBeforeSecondMigration string
	if err := database.QueryRow("SELECT hashed_password FROM users WHERE id = 1").Scan(&hashBeforeSecondMigration); err != nil {
		database.Close()
		t.Fatalf("read migrated password: %v", err)
	}
	if err := RunMigrations(); err != nil {
		t.Fatalf("second RunMigrations() returned error: %v", err)
	}

	database, err = getDB()
	if err != nil {
		t.Fatalf("reopen test database: %v", err)
	}
	var hashAfterSecondMigration string
	if err := database.QueryRow("SELECT hashed_password FROM users WHERE id = 1").Scan(&hashAfterSecondMigration); err != nil {
		t.Fatalf("read password after second migration: %v", err)
	}
	if hashAfterSecondMigration != hashBeforeSecondMigration {
		t.Fatal("MigratePasswords() rehashed an already migrated password")
	}
}

func TestMigratePasswordsLeavesNullPasswordsAlone(t *testing.T) {
	useTestDatabase(t)

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	_, err = database.Exec("INSERT INTO users (id, name, hashed_password) VALUES ($1, $2, NULL)", 2, "no-password")
	if err != nil {
		t.Fatalf("insert null password: %v", err)
	}

	if err := RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}

	database, err = getDB()
	if err != nil {
		t.Fatalf("reopen test database: %v", err)
	}
	var password sql.NullString
	if err := database.QueryRow("SELECT hashed_password FROM users WHERE id = 2").Scan(&password); err != nil {
		t.Fatalf("read null password: %v", err)
	}
	if password.Valid {
		t.Fatalf("MigratePasswords() changed a NULL password to %q", password.String)
	}
}

func TestRefreshTokenQueries(t *testing.T) {
	useTestDatabase(t)

	expiresAt := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	token := models.RefreshToken{ID: "refresh-token", UserID: 1, ExpiresAt: expiresAt}
	if err := StoreRefreshToken(token); err != nil {
		t.Fatalf("StoreRefreshToken() returned error: %v", err)
	}

	stored, err := GetRefreshToken(token.ID)
	if err != nil {
		t.Fatalf("GetRefreshToken() returned error: %v", err)
	}
	if stored.ID != token.ID || stored.UserID != token.UserID || !stored.ExpiresAt.Equal(expiresAt) || stored.CreatedAt.IsZero() {
		t.Fatalf("GetRefreshToken() = %#v, want stored token", stored)
	}

	if err := DeleteRefreshToken(token.ID); err != nil {
		t.Fatalf("DeleteRefreshToken() returned error: %v", err)
	}
	if _, err := GetRefreshToken(token.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetRefreshToken() after deletion returned %v, want sql.ErrNoRows", err)
	}
}

func TestWebAuthnCredentialQueries(t *testing.T) {
	useTestDatabase(t)

	credential := webauthn.Credential{ID: []byte{0x01, 0x02, 0x03}}
	if err := AddWebAuthnCredential(1, credential, "Laptop"); err != nil {
		t.Fatalf("AddWebAuthnCredential() returned error: %v", err)
	}

	userID, err := GetWebAuthnUserIDByCredentialID(credential.ID)
	if err != nil || userID != 1 {
		t.Fatalf("GetWebAuthnUserIDByCredentialID() = (%d, %v), want (1, nil)", userID, err)
	}
	credentials, err := GetWebAuthnCredentialsByUserID(1)
	if err != nil || len(credentials) != 1 || string(credentials[0].ID) != string(credential.ID) {
		t.Fatalf("GetWebAuthnCredentialsByUserID() = (%#v, %v), want credential", credentials, err)
	}
	infos, err := ListUserWebAuthnCredentials(1)
	if err != nil || len(infos) != 1 || infos[0].Name != "Laptop" || infos[0].CreatedAt.IsZero() {
		t.Fatalf("ListUserWebAuthnCredentials() = (%#v, %v), want credential info", infos, err)
	}

	deleted, err := DeleteWebAuthnCredential(credential.ID, 1)
	if err != nil || !deleted {
		t.Fatalf("DeleteWebAuthnCredential() = (%t, %v), want (true, nil)", deleted, err)
	}
}

func useTestDatabase(t *testing.T) {
	t.Helper()

	database, databaseURL := testdb.Start(t)
	if err := Configure(databaseURL); err != nil {
		t.Fatalf("configure test database: %v", err)
	}
	t.Cleanup(func() { Close() })
	if err := RunMigrations(); err != nil {
		t.Fatalf("initialize test database: %v", err)
	}
	if _, err := database.Exec("INSERT INTO users (id, name, hashed_password) VALUES (1, 'reader', 'secret')"); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
}
