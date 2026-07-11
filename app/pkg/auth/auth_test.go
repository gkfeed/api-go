package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gkfeed/api/internal/models"
)

func TestBasicAuthProvidesAuthenticatedUser(t *testing.T) {
	replaceUserLookup(t, func(name string) (models.User, error) {
		return models.User{ID: 7, Name: name, HashedPassword: "secret"}, nil
	})

	var receivedUser models.User
	handler := BasicAuth(func(w http.ResponseWriter, r *http.Request) {
		receivedUser, _ = UserFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("reader", "secret")
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if receivedUser.ID != 7 || receivedUser.Name != "reader" {
		t.Fatalf("authenticated user = %#v, want reader with ID 7", receivedUser)
	}
}

func TestBasicAuthRejectsInvalidPassword(t *testing.T) {
	replaceUserLookup(t, func(name string) (models.User, error) {
		return models.User{Name: name, HashedPassword: "secret"}, nil
	})

	handler := BasicAuth(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("reader", "wrong")
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestBasicAuthReturnsServerErrorWhenLookupFails(t *testing.T) {
	replaceUserLookup(t, func(string) (models.User, error) {
		return models.User{}, errors.New("database unavailable")
	})

	handler := BasicAuth(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("reader", "secret")
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func replaceUserLookup(t *testing.T, lookup func(string) (models.User, error)) {
	t.Helper()
	original := getUser
	getUser = lookup
	t.Cleanup(func() { getUser = original })
}
