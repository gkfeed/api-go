package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	addressEnvironmentVariable           = "GKFEED_ADDRESS"
	databaseEnvironmentVariable          = "GKFEED_DB_PATH"
	allowedOriginsEnvironmentVariable    = "GKFEED_ALLOWED_ORIGINS"
	jwtSecretEnvironmentVariable         = "GKFEED_JWT_SECRET"
	jwtAccessTTLEnvironmentVariable      = "GKFEED_JWT_ACCESS_TTL"
	jwtRefreshTTLEnvironmentVariable     = "GKFEED_JWT_REFRESH_TTL"
	webauthnRPIDEnvironmentVariable      = "GKFEED_WEBAUTHN_RP_ID"
	webauthnRPOriginEnvironmentVariable  = "GKFEED_WEBAUTHN_RP_ORIGIN"
	webauthnRPDisplayEnvironmentVariable = "GKFEED_WEBAUTHN_RP_DISPLAY"
)

var defaultAllowedOrigins = []string{
	"http://localhost",
	"http://localhost:4200",
	"http://localhost:8086",
}

const (
	defaultJWTSecret         = "change-me-in-production"
	defaultAccessTokenTTL    = 15 * time.Minute
	defaultRefreshTokenTTL   = 720 * time.Hour
	defaultWebAuthnRPDisplay = "GKFeed"
)

type Config struct {
	Address           string
	DatabasePath      string
	AllowedOrigins    []string
	ReadHeaderTimeout time.Duration

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	WebAuthnRPID      string
	WebAuthnRPOrigin  string
	WebAuthnRPDisplay string
}

func Load() Config {
	return Config{
		Address:           valueOrDefault(addressEnvironmentVariable, ":8086"),
		DatabasePath:      valueOrDefault(databaseEnvironmentVariable, filepath.Join("..", "data", "db.sqlite")),
		AllowedOrigins:    allowedOrigins(),
		ReadHeaderTimeout: 5 * time.Second,

		JWTSecret:       valueOrDefault(jwtSecretEnvironmentVariable, defaultJWTSecret),
		AccessTokenTTL:  durationOrDefault(jwtAccessTTLEnvironmentVariable, defaultAccessTokenTTL),
		RefreshTokenTTL: durationOrDefault(jwtRefreshTTLEnvironmentVariable, defaultRefreshTokenTTL),

		WebAuthnRPID:      os.Getenv(webauthnRPIDEnvironmentVariable),
		WebAuthnRPOrigin:  os.Getenv(webauthnRPOriginEnvironmentVariable),
		WebAuthnRPDisplay: valueOrDefault(webauthnRPDisplayEnvironmentVariable, defaultWebAuthnRPDisplay),
	}
}

func allowedOrigins() []string {
	value := os.Getenv(allowedOriginsEnvironmentVariable)
	if value == "" {
		return slices.Clone(defaultAllowedOrigins)
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

func durationOrDefault(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}
