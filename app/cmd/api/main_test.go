package main

import (
	"net/http"
	"net/http/httptest"
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
