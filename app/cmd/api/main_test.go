package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"gkfeed/api/internal/config"
)

func TestDeleteRouteDoesNotAllowGet(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/delete?id=1", nil)
	response := httptest.NewRecorder()

	newHandler(config.Config{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("GET /api/v1/delete returned status %d; want %d", response.Code, http.StatusNotFound)
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
