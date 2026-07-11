package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueryID(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantID     int
		wantOK     bool
		wantStatus int
	}{
		{name: "valid", query: "?id=42", wantID: 42, wantOK: true, wantStatus: http.StatusOK},
		{name: "missing", wantOK: false, wantStatus: http.StatusBadRequest},
		{name: "not a number", query: "?id=nope", wantOK: false, wantStatus: http.StatusBadRequest},
		{name: "not positive", query: "?id=0", wantOK: false, wantStatus: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/item"+test.query, nil)
			response := httptest.NewRecorder()

			id, ok := queryID(response, request)

			if id != test.wantID || ok != test.wantOK {
				t.Fatalf("queryID() = (%d, %t), want (%d, %t)", id, ok, test.wantID, test.wantOK)
			}
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestDecodeJSONRejectsUnknownFieldsAndMultipleValues(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"name":"example","extra":true}`},
		{name: "multiple values", body: `{"name":"first"} {"name":"second"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			var input struct {
				Name string `json:"name"`
			}

			if decodeJSON(response, request, &input) {
				t.Fatal("decodeJSON() accepted invalid input")
			}
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}
