package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"gkfeed/api/internal/config"
	"gkfeed/api/internal/db"
)

func TestDeleteRouteDoesNotAllowGet(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/delete?id=1", nil)
	response := httptest.NewRecorder()

	newHandler(config.Config{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("GET /api/v1/delete returned status %d; want %d", response.Code, http.StatusNotFound)
	}
}

func TestMeRouteAcceptsBasicAuth(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "db.sqlite")
	db.Configure(databasePath)
	t.Cleanup(func() { db.Configure("") })

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}

	database, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(
		"INSERT INTO users (id, name, password) VALUES (?, ?, ?)",
		7, "reader", "secret",
	); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() after legacy user insert returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.SetBasicAuth("reader", "secret")
	response := httptest.NewRecorder()

	newHandler(config.Config{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/auth/me returned status %d; want %d", response.Code, http.StatusOK)
	}

	var user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 7 || user.Name != "reader" {
		t.Fatalf("GET /api/v1/auth/me returned %#v; want reader with ID 7", user)
	}
}

func TestItemRouteRequiresAuthenticationAndOwner(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "db.sqlite")
	db.Configure(databasePath)
	t.Cleanup(func() { db.Configure("") })

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() returned error: %v", err)
	}

	database, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer database.Close()

	for _, user := range []struct {
		id       int
		name     string
		password string
	}{
		{1, "owner", "owner-password"},
		{2, "other", "other-password"},
	} {
		if _, err := database.Exec("INSERT INTO users (id, name, password) VALUES (?, ?, ?)", user.id, user.name, user.password); err != nil {
			t.Fatalf("insert test user %d: %v", user.id, err)
		}
	}
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() after legacy user inserts returned error: %v", err)
	}
	if _, err := database.Exec(
		"INSERT INTO feed (id, title, url, type, user_id) VALUES (?, ?, ?, ?, ?)",
		11, "Owner feed", "https://example.com/feed", "web", 1,
	); err != nil {
		t.Fatalf("insert test feed: %v", err)
	}
	if _, err := database.Exec(
		"INSERT INTO item (id, feed_id, title, text, date, link) VALUES (?, ?, ?, ?, ?, ?)",
		22, 11, "Owner item", "private", "2026-01-01T00:00:00Z", "https://example.com/item",
	); err != nil {
		t.Fatalf("insert test item: %v", err)
	}

	tests := []struct {
		name       string
		username   string
		password   string
		wantStatus int
	}{
		{name: "requires authentication", wantStatus: http.StatusUnauthorized},
		{name: "allows owner", username: "owner", password: "owner-password", wantStatus: http.StatusOK},
		{name: "rejects another user", username: "other", password: "other-password", wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/item?id=22", nil)
			if test.username != "" {
				request.SetBasicAuth(test.username, test.password)
			}
			response := httptest.NewRecorder()

			newHandler(config.Config{}).ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("GET /api/v1/item returned status %d; want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestFeedTypesRouteIsPublic(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed_types", nil)
	response := httptest.NewRecorder()

	newHandler(config.Config{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/feed_types returned status %d; want %d", response.Code, http.StatusOK)
	}

	var feedTypes []string
	if err := json.NewDecoder(response.Body).Decode(&feedTypes); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !slices.Contains(feedTypes, "web") || !slices.Contains(feedTypes, "spoti:playlist") {
		t.Fatalf("GET /api/v1/feed_types returned unexpected feed types: %#v", feedTypes)
	}
}
