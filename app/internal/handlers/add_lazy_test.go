package handlers

import (
	"bytes"
	"encoding/json"
	"gkfeed/api/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock the db and services dependencies for testing
// In a real project, you might use a mocking library or interfaces.

// Mock for db.GetUserFromDB
func mockGetUserFromDB(username string) models.User {
	return models.User{ID: 1, Name: username}
}

// Mock for db.AddFeed
func mockAddFeed(feed models.Feed, userID int) models.Feed {
	feed.ID = 1 // Assign a dummy ID
	return feed
}

// Mock for services.FeedFactory.CreateFromUrl
func mockCreateFromUrl(url string) (*models.Feed, error) {
	return &models.Feed{
			Title: "Test Feed",
			Type:  "rss",
			Url:   url,
		},
		nil
}

func TestHandleAddFeedLazy(t *testing.T) {
	// Override the actual functions with our mocks for testing
	originalGetUserFromDB := dbGetUserFromDB
	originalAddFeed := dbAddFeed
	originalCreateFromUrl := servicesCreateFromUrl

	dbGetUserFromDB = mockGetUserFromDB
	dbAddFeed = mockAddFeed
	servicesCreateFromUrl = mockCreateFromUrl

	defer func() {
		// Restore original functions after the test
		dbGetUserFromDB = originalGetUserFromDB
		dbAddFeed = originalAddFeed
		servicesCreateFromUrl = originalCreateFromUrl
	}()

	testURL := "https://hdrezka.me/series/thriller/41647-igra-v-kalmara-2021-latest.html"
	requestBody, _ := json.Marshal(map[string]string{"url": testURL})

	req, err := http.NewRequest("POST", "/add-lazy", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("testuser", "testpassword")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleAddFeedLazy)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"created":true,"item":{"id":1,"title":"Test Feed","type":"rss","url":"https://hdrezka.me/series/thriller/41647-igra-v-kalmara-2021-latest.html","userid":0}}`
	// Unmarshal and then marshal again to handle potential differences in whitespace/order
	var actual map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &actual)
	actualJSON, _ := json.Marshal(actual)

	var expectedMap map[string]interface{}
	json.Unmarshal([]byte(expected), &expectedMap)
	expectedJSON, _ := json.Marshal(expectedMap)

	if string(actualJSON) != string(expectedJSON) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
