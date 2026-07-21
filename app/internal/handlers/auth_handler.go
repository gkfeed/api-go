package handlers

import (
	"gkfeed/api/internal/auth"
	"gkfeed/api/internal/config"
)

type AuthHandler struct {
	cfg             config.Config
	webAuthnService *auth.WebAuthnService
}

func NewAuthHandler(cfg config.Config, webAuthnService *auth.WebAuthnService) *AuthHandler {
	return &AuthHandler{cfg: cfg, webAuthnService: webAuthnService}
}
