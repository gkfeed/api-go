package config

import (
	"reflect"
	"testing"
)

func TestLoadUsesEnvironment(t *testing.T) {
	t.Setenv(addressEnvironmentVariable, "127.0.0.1:9000")
	t.Setenv(databaseEnvironmentVariable, "/tmp/gkfeed.sqlite")
	t.Setenv(allowedOriginsEnvironmentVariable, "https://one.example, https://two.example")

	configuration := Load()

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
}

func TestLoadReturnsIndependentDefaultOrigins(t *testing.T) {
	t.Setenv(allowedOriginsEnvironmentVariable, "")
	first := Load()
	first.AllowedOrigins[0] = "changed"

	second := Load()
	if second.AllowedOrigins[0] == "changed" {
		t.Fatal("Load() returned shared default origins")
	}
}
