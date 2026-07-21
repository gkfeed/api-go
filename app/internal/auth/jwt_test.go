package auth

import (
	"testing"
	"time"

	"gkfeed/api/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateAccessTokenRejectsOtherHMACAlgorithms(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret"}
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "42",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		Name: "test user",
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatalf("sign HS384 token: %v", err)
	}

	if _, err := ValidateAccessToken(token, cfg); err == nil {
		t.Fatal("ValidateAccessToken() accepted HS384, want only HS256")
	}
}
