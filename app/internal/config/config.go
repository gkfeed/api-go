package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	addressEnvironmentVariable        = "GKFEED_ADDRESS"
	databaseEnvironmentVariable       = "GKFEED_DB_PATH"
	allowedOriginsEnvironmentVariable = "GKFEED_ALLOWED_ORIGINS"
)

var defaultAllowedOrigins = []string{
	"http://localhost",
	"http://localhost:4200",
	"http://localhost:8086",
}

type Config struct {
	Address           string
	DatabasePath      string
	AllowedOrigins    []string
	ReadHeaderTimeout time.Duration
}

func Load() Config {
	return Config{
		Address:           valueOrDefault(addressEnvironmentVariable, ":8086"),
		DatabasePath:      valueOrDefault(databaseEnvironmentVariable, filepath.Join("..", "data", "db.sqlite")),
		AllowedOrigins:    allowedOrigins(),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func allowedOrigins() []string {
	value := os.Getenv(allowedOriginsEnvironmentVariable)
	if value == "" {
		return append([]string(nil), defaultAllowedOrigins...)
	}

	origins := make([]string, 0)
	for origin := range strings.SplitSeq(value, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
