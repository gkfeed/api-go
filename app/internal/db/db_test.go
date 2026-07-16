package db

import (
	"path/filepath"
	"testing"
	"time"

	"gkfeed/api/internal/models"
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
		"INSERT INTO item (feed_id, title, text, date, link) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)",
		feed.ID, "First", "first", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "https://example.com/first",
		feed.ID, "Second", "second", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), "https://example.com/second",
	)
	database.Close()
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
}

func useTestDatabase(t *testing.T) {
	t.Helper()

	originalPath := dbPath
	dbPath = filepath.Join(t.TempDir(), "db.sqlite")
	t.Cleanup(func() { dbPath = originalPath })

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	schema := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, password TEXT)",
		"CREATE TABLE feed (id INTEGER PRIMARY KEY, title TEXT, url TEXT, type TEXT, user_id INTEGER)",
		"CREATE TABLE item (id INTEGER PRIMARY KEY, feed_id INTEGER, title TEXT, text TEXT, date DATETIME, link TEXT)",
		"CREATE TABLE deleted_items (user_id INTEGER, item_id INTEGER)",
		"CREATE TABLE webauthn_credentials (id BLOB PRIMARY KEY, user_id INTEGER NOT NULL, credential TEXT NOT NULL, name TEXT NOT NULL DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, last_used_at DATETIME)",
		"CREATE TABLE refresh_tokens (id TEXT PRIMARY KEY, user_id INTEGER NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
		"INSERT INTO users (id, name, password) VALUES (1, 'reader', 'secret')",
	}
	for _, statement := range schema {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("initialize test database: %v", err)
		}
	}
}
