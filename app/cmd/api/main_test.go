package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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

func TestSwaggerRoutesAreUnderAPI(t *testing.T) {
	for _, path := range []string{"/api/swagger/index.html", "/api/swagger/doc.json"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()

		newHandler(config.Config{}).ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("GET %s returned status %d; want %d", path, response.Code, http.StatusOK)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	response := httptest.NewRecorder()
	newHandler(config.Config{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("GET /swagger/index.html returned status %d; want %d", response.Code, http.StatusNotFound)
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
		"INSERT INTO users (id, name, hashed_password) VALUES (?, ?, ?)",
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
		if _, err := database.Exec("INSERT INTO users (id, name, hashed_password) VALUES (?, ?, ?)", user.id, user.name, user.password); err != nil {
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

func TestInboxRoutes(t *testing.T) {
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
	_, err = database.Exec(`
		INSERT INTO users (id, name, hashed_password) VALUES
			(1, 'sender', 'sender-password'),
			(2, 'recipient', 'recipient-password');
		INSERT INTO feed (id, title, url, type, user_id)
			VALUES (10, 'Source', 'https://example.com/feed', 'web', 1);
		INSERT INTO item (id, feed_id, title, text, date, link)
			VALUES (20, 10, 'An item', 'Useful text', '2026-08-10T10:00:00Z', 'https://example.com/item')`)
	database.Close()
	if err != nil {
		t.Fatalf("insert test fixtures: %v", err)
	}
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("RunMigrations() after user insert returned error: %v", err)
	}

	handler := newHandler(config.Config{})
	doRequest := func(method, path, username, password, body, key string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.SetBasicAuth(username, password)
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	shareBody := `{"item_id":20,"recipient_user_id":2,"note":"Read this"}`
	response := doRequest(http.MethodPost, "/api/v1/shares", "sender", "sender-password", shareBody, "share-route-test")
	if response.Code != http.StatusNotFound {
		t.Fatalf("share before Inbox creation status = %d, want %d", response.Code, http.StatusNotFound)
	}

	response = doRequest(http.MethodPost, "/api/v1/feeds/me/inbox", "recipient", "recipient-password", "", "")
	if response.Code != http.StatusCreated {
		t.Fatalf("create Inbox status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var createdInbox struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&createdInbox); err != nil {
		t.Fatalf("decode Inbox creation response: %v", err)
	}
	response = doRequest(http.MethodDelete, "/api/v1/delete?id="+strconv.Itoa(createdInbox.ID), "recipient", "recipient-password", "", "")
	if response.Code != http.StatusConflict {
		t.Fatalf("delete Inbox status = %d, want %d", response.Code, http.StatusConflict)
	}
	response = doRequest(http.MethodPost, "/api/v1/feeds/me/inbox", "recipient", "recipient-password", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("repeat Inbox creation status = %d, want %d", response.Code, http.StatusOK)
	}

	response = doRequest(http.MethodPost, "/api/v1/shares", "sender", "sender-password", shareBody, "share-route-test")
	if response.Code != http.StatusCreated {
		t.Fatalf("share status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var createdDelivery struct {
		ID     string `json:"delivery_id"`
		ItemID int    `json:"item_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&createdDelivery); err != nil {
		t.Fatalf("decode share response: %v", err)
	}
	if createdDelivery.ID == "" || createdDelivery.ItemID == 0 || createdDelivery.ItemID == 20 {
		t.Fatalf("share response = %#v, want delivery with a cloned item", createdDelivery)
	}

	response = doRequest(http.MethodPost, "/api/v1/shares", "sender", "sender-password", shareBody, "share-route-test")
	if response.Code != http.StatusOK {
		t.Fatalf("share retry status = %d, want %d", response.Code, http.StatusOK)
	}
	var retryDelivery struct {
		ID string `json:"delivery_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&retryDelivery); err != nil {
		t.Fatalf("decode retry response: %v", err)
	}
	if retryDelivery.ID != createdDelivery.ID {
		t.Fatalf("share retry delivery ID = %q, want %q", retryDelivery.ID, createdDelivery.ID)
	}

	response = doRequest(http.MethodGet, "/api/v1/feeds/me/inbox/items?state=unread", "recipient", "recipient-password", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("list Inbox status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var inbox struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&inbox); err != nil {
		t.Fatalf("decode Inbox response: %v", err)
	}
	if len(inbox.Items) != 1 {
		t.Fatalf("Inbox contains %d items, want 1", len(inbox.Items))
	}

	itemPath := "/api/v1/inbox/items/" + strconv.Itoa(createdDelivery.ItemID)
	response = doRequest(http.MethodPost, itemPath+"/read", "sender", "sender-password", "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("sender reading recipient item status = %d, want %d", response.Code, http.StatusNotFound)
	}
	response = doRequest(http.MethodPost, itemPath+"/read", "recipient", "recipient-password", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("mark Inbox item read status = %d, want %d", response.Code, http.StatusOK)
	}
	response = doRequest(http.MethodPost, itemPath+"/archive", "recipient", "recipient-password", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("archive Inbox item status = %d, want %d", response.Code, http.StatusOK)
	}
}
