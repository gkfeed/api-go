package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoadUsesEnvironment(t *testing.T) {
	const jwtSecret = "a-random-secret-with-at-least-32-bytes"
	t.Setenv(addressEnvironmentVariable, "127.0.0.1:9000")
	t.Setenv(databaseEnvironmentVariable, "/tmp/gkfeed.sqlite")
	t.Setenv(allowedOriginsEnvironmentVariable, "https://one.example, https://two.example")
	t.Setenv(jwtSecretEnvironmentVariable, jwtSecret)

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if configuration.Address != "127.0.0.1:9000" {
		t.Fatalf("Address = %q, want %q", configuration.Address, "127.0.0.1:9000")
	}
	if configuration.DatabasePath != "/tmp/gkfeed.sqlite" {
		t.Fatalf("DatabasePath = %q, want %q", configuration.DatabasePath, "/tmp/gkfeed.sqlite")
	}
	wantOrigins := []string{"https://one.example", "https://two.example"}
	if !reflect.DeepEqual(configuration.AllowedOrigins, wantOrigins) {
		t.Fatalf("AllowedOrigins = %#v, want %#v", configuration.AllowedOrigins, wantOrigins)
	}
	if configuration.JWTSecret != jwtSecret {
		t.Fatalf("JWTSecret = %q, want configured secret", configuration.JWTSecret)
	}
}

func TestLoadReturnsIndependentDefaultOrigins(t *testing.T) {
	t.Setenv(allowedOriginsEnvironmentVariable, "")
	t.Setenv(jwtSecretEnvironmentVariable, "a-random-secret-with-at-least-32-bytes")
	first, err := Load()
	if err != nil {
		t.Fatalf("first Load() returned error: %v", err)
	}
	first.AllowedOrigins[0] = "changed"

	second, err := Load()
	if err != nil {
		t.Fatalf("second Load() returned error: %v", err)
	}
	if second.AllowedOrigins[0] == "changed" {
		t.Fatal("Load() returned shared default origins")
	}
}

func TestLoadRejectsMissingJWTSecret(t *testing.T) {
	t.Setenv(jwtSecretEnvironmentVariable, "")

	_, err := Load()

	if err == nil {
		t.Fatal("Load() returned nil error without a JWT secret")
	}
	if !strings.Contains(err.Error(), jwtSecretEnvironmentVariable) {
		t.Fatalf("Load() error = %q, want it to name %s", err, jwtSecretEnvironmentVariable)
	}
}

func TestLoadRejectsShortJWTSecret(t *testing.T) {
	t.Setenv(jwtSecretEnvironmentVariable, strings.Repeat("x", minJWTSecretLength-1))

	_, err := Load()

	if err == nil {
		t.Fatal("Load() returned nil error for a short JWT secret")
	}
}

func TestLoadAcceptsMinimumLengthJWTSecret(t *testing.T) {
	t.Setenv(jwtSecretEnvironmentVariable, strings.Repeat("x", minJWTSecretLength))

	configuration, err := Load()

	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(configuration.JWTSecret) != minJWTSecretLength {
		t.Fatalf("JWTSecret length = %d, want %d", len(configuration.JWTSecret), minJWTSecretLength)
	}
}
