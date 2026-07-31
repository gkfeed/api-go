package auth

import (
	"strings"
	"testing"

	"gkfeed/api/internal/config"
)

func TestNewWebAuthnServiceRequiresRPConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.Config
		wantEnvVar string
	}{
		{
			name:       "missing RP ID",
			cfg:        config.Config{WebAuthnRPOrigin: "http://localhost:8086"},
			wantEnvVar: "GKFEED_WEBAUTHN_RP_ID",
		},
		{
			name:       "missing RP origin",
			cfg:        config.Config{WebAuthnRPID: "localhost"},
			wantEnvVar: "GKFEED_WEBAUTHN_RP_ORIGIN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWebAuthnService(tt.cfg)
			if err == nil {
				t.Fatal("NewWebAuthnService() returned nil error")
			}
			if !strings.Contains(err.Error(), tt.wantEnvVar) {
				t.Fatalf("NewWebAuthnService() error = %q, want it to mention %s", err, tt.wantEnvVar)
			}
		})
	}
}
