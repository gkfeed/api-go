package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gkfeed/api/internal/models"
	"gkfeed/api/pkg/auth"
)

func mockAddFeed(feed models.Feed, userID int) (models.Feed, error) {
	feed.ID = 1
	feed.UserID = userID
	return feed, nil
}

func mockCreateFromURL(url string) (models.Feed, error) {
	return models.Feed{
			Title: "Test Feed",
			Type:  "rss",
			URL:   url,
		},
		nil
}

func TestHandleAddFeedLazy(t *testing.T) {
	originalAddFeed := dbAddFeed
	originalCreateFromURL := servicesCreateFromURL

	dbAddFeed = mockAddFeed
	servicesCreateFromURL = mockCreateFromURL

	defer func() {
		dbAddFeed = originalAddFeed
		servicesCreateFromURL = originalCreateFromURL
	}()

	const testURL = "https://hdrezka.me/series/thriller/41647-igra-v-kalmara-2021-latest.html"
	request := httptest.NewRequest(http.MethodPost, "/add-lazy", bytes.NewBufferString(`{"url":"`+testURL+`"}`))
	request = request.WithContext(auth.WithUser(request.Context(), models.User{ID: 1, Name: "testuser"}))
	response := httptest.NewRecorder()

	HandleAddFeedLazy(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body feedMutationResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Created || body.Item.ID != 1 || body.Item.UserID != 1 || body.Item.URL != testURL {
		t.Fatalf("response = %#v, want the created feed", body)
	}
}

func TestHandleAddFeedLazyReturnsServerErrorWhenInsertFails(t *testing.T) {
	originalAddFeed := dbAddFeed
	originalCreateFromURL := servicesCreateFromURL
	t.Cleanup(func() {
		dbAddFeed = originalAddFeed
		servicesCreateFromURL = originalCreateFromURL
	})

	dbAddFeed = func(models.Feed, int) (models.Feed, error) {
		return models.Feed{}, errors.New("database unavailable")
	}
	servicesCreateFromURL = mockCreateFromURL

	request := httptest.NewRequest(http.MethodPost, "/add-lazy", bytes.NewBufferString(`{"url":"https://example.com"}`))
	request = request.WithContext(auth.WithUser(request.Context(), models.User{ID: 1, Name: "testuser"}))
	response := httptest.NewRecorder()

	HandleAddFeedLazy(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
