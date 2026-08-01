package config

import (
	"reflect"
	"testing"
)

func TestLoadUsesEnvironment(t *testing.T) {
	t.Setenv(addressEnvironmentVariable, "127.0.0.1:9000")
	t.Setenv(databaseEnvironmentVariable, "/tmp/gkfeed.sqlite")
	t.Setenv(allowedOriginsEnvironmentVariable, "https://one.example, https://two.example")
	t.Setenv(accessTTLEnvironmentVariable, "45m")

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
	if configuration.AccessTokenTTL.String() != "45m0s" {
		t.Fatalf("AccessTokenTTL = %s, want 45m0s", configuration.AccessTokenTTL)
	}
}

func TestLoadReturnsIndependentDefaultOrigins(t *testing.T) {
	t.Setenv(allowedOriginsEnvironmentVariable, "")
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
