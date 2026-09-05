package auth

import (
	"bytes"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"gkfeed/api/internal/config"
	"gkfeed/api/internal/models"
	"gkfeed/api/internal/passwordhash"
)

func TestBasicAuthProvidesAuthenticatedUser(t *testing.T) {
	hash := testPasswordHash(t, "secret")
	replaceUserLookup(t, func(name string) (models.User, error) {
		return models.User{ID: 7, Name: name, HashedPassword: sql.NullString{String: hash, Valid: true}}, nil
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
	hash := testPasswordHash(t, "secret")
	replaceUserLookup(t, func(name string) (models.User, error) {
		return models.User{Name: name, HashedPassword: sql.NullString{String: hash, Valid: true}}, nil
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

func TestAuthenticateJWT(t *testing.T) {
	cfg := config.Config{
		JWTSecret:      "test-secret",
		AccessTokenTTL: 15 * time.Minute,
	}

	token, err := GenerateAccessToken(42, "jwtuser", cfg)
	if err != nil {
		t.Fatalf("GenerateAccessToken() returned error: %v", err)
	}

	var receivedUser models.User
	handler := Authenticate(cfg)(func(w http.ResponseWriter, r *http.Request) {
		receivedUser, _ = UserFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if receivedUser.ID != 42 || receivedUser.Name != "jwtuser" {
		t.Fatalf("authenticated user = %#v, want jwtuser with ID 42", receivedUser)
	}
}

func TestAuthenticateFallsBackToBasicAuth(t *testing.T) {
	hash := testPasswordHash(t, "secret")
	replaceUserLookup(t, func(name string) (models.User, error) {
		return models.User{ID: 99, Name: name, HashedPassword: sql.NullString{String: hash, Valid: true}}, nil
	})

	cfg := config.Config{JWTSecret: "test-secret"}

	var receivedUser models.User
	handler := Authenticate(cfg)(func(w http.ResponseWriter, r *http.Request) {
		receivedUser, _ = UserFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("basicuser", "secret")
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if receivedUser.ID != 99 || receivedUser.Name != "basicuser" {
		t.Fatalf("authenticated user = %#v, want basicuser with ID 99", receivedUser)
	}
}

func TestAuthenticateRejectsInvalidJWT(t *testing.T) {
	replaceUserLookup(t, func(string) (models.User, error) {
		return models.User{}, errors.New("no user")
	})

	cfg := config.Config{JWTSecret: "test-secret"}

	handler := Authenticate(cfg)(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})

	t.Run("invalid token", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Authorization", "Bearer invalid.jwt.token")
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		validCfg := config.Config{JWTSecret: "correct-secret", AccessTokenTTL: 15 * time.Minute}
		token, err := GenerateAccessToken(1, "user", validCfg)
		if err != nil {
			t.Fatalf("GenerateAccessToken() returned error: %v", err)
		}
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
	})
}

func TestAuthenticateRejectsWithoutCredentials(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret"}

	handler := Authenticate(cfg)(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticateDoesNotLogAuthorizationHeader(t *testing.T) {
	replaceUserLookup(t, func(name string) (models.User, error) {
		return models.User{Name: name, HashedPassword: sql.NullString{String: "secret", Valid: true}}, nil
	})

	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
	})

	handler := Authenticate(config.Config{JWTSecret: "test-secret"})(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("reader", "a-very-long-password-that-must-not-be-logged")
	authHeader := request.Header.Get("Authorization")
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	output := logs.String()
	if strings.Contains(output, authHeader) {
		t.Fatalf("log output contains Authorization header: %q", output)
	}
	if strings.Contains(output, authHeader[:min(len(authHeader), 30)]) {
		t.Fatalf("log output contains part of Authorization header: %q", output)
	}
}

func TestJWTAuthRejectsBasicAuth(t *testing.T) {
	cfg := config.Config{
		JWTSecret:      "test-secret",
		AccessTokenTTL: 15 * time.Minute,
	}

	token, err := GenerateAccessToken(1, "user", cfg)
	if err != nil {
		t.Fatalf("GenerateAccessToken() returned error: %v", err)
	}

	handler := JWTAuth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("accepts valid JWT", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
		}
	})

	t.Run("rejects Basic Auth", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.SetBasicAuth("user", "password")
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
	})
}

func TestJWTAuthRejectsExpiredToken(t *testing.T) {
	cfg := config.Config{
		JWTSecret:      "test-secret",
		AccessTokenTTL: -1 * time.Hour,
	}

	token, err := GenerateAccessToken(1, "user", cfg)
	if err != nil {
		t.Fatalf("GenerateAccessToken() returned error: %v", err)
	}

	handler := JWTAuth(cfg)(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	cfg := config.Config{
		JWTSecret:      "test-secret",
		AccessTokenTTL: 15 * time.Minute,
	}

	token, err := GenerateAccessToken(42, "testuser", cfg)
	if err != nil {
		t.Fatalf("GenerateAccessToken() returned error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateAccessToken() returned empty token")
	}

	claims, err := ValidateAccessToken(token, cfg)
	if err != nil {
		t.Fatalf("ValidateAccessToken() returned error: %v", err)
	}
	if claims.Subject != strconv.Itoa(42) {
		t.Fatalf("claims.Subject = %q, want %q", claims.Subject, "42")
	}
	if claims.Name != "testuser" {
		t.Fatalf("claims.Name = %q, want %q", claims.Name, "testuser")
	}
}

func TestValidateAccessTokenRejectsWrongSecret(t *testing.T) {
	cfg := config.Config{
		JWTSecret:      "test-secret",
		AccessTokenTTL: 15 * time.Minute,
	}
	token, err := GenerateAccessToken(42, "user", cfg)
	if err != nil {
		t.Fatalf("GenerateAccessToken() returned error: %v", err)
	}

	otherCfg := config.Config{
		JWTSecret: "different-secret",
	}
	_, err = ValidateAccessToken(token, otherCfg)
	if err == nil {
		t.Fatal("ValidateAccessToken() should have returned error for wrong secret")
	}
}

func replaceUserLookup(t *testing.T, lookup func(string) (models.User, error)) {
	t.Helper()
	original := getUser
	getUser = lookup
	t.Cleanup(func() { getUser = original })
}

func testPasswordHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := passwordhash.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() returned error: %v", err)
	}
	return hash
}

func TestBasicAuthRejectsNullPassword(t *testing.T) {
	replaceUserLookup(t, func(name string) (models.User, error) {
		return models.User{ID: 7, Name: name}, nil
	})
	for _, password := range []string{"", "secret"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.SetBasicAuth("passwordless", password)
		response := httptest.NewRecorder()
		BasicAuth(func(http.ResponseWriter, *http.Request) { t.Error("authenticated passwordless user") })(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", response.Code)
		}
	}
}
