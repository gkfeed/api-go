package testdb

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const databaseURLEnvironmentVariable = "GKFEED_TEST_DATABASE_URL"

// Start creates an isolated PostgreSQL schema for an integration test. Tests
// are skipped when GKFEED_TEST_DATABASE_URL is not set.
func Start(t testing.TB) (*sql.DB, string) {
	t.Helper()

	databaseURL := os.Getenv(databaseURLEnvironmentVariable)
	if databaseURL == "" {
		t.Skipf("set %s to run PostgreSQL integration tests", databaseURLEnvironmentVariable)
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse %s: %v", databaseURLEnvironmentVariable, err)
	}

	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		t.Fatalf("generate test schema name: %v", err)
	}
	schema := "gkfeed_test_" + hex.EncodeToString(random)

	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open test PostgreSQL database: %v", err)
	}
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		admin.Close()
		t.Fatalf("create test schema: %v", err)
	}

	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()
	isolatedURL := parsedURL.String()

	database, err := sql.Open("pgx", isolatedURL)
	if err != nil {
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
		t.Fatalf("open isolated test database: %v", err)
	}
	if err := database.Ping(); err != nil {
		database.Close()
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
		t.Fatalf("ping isolated test database: %v", err)
	}

	t.Cleanup(func() {
		database.Close()
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})
	return database, isolatedURL
}
