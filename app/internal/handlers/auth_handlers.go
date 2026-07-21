package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	authsvc "gkfeed/api/internal/auth"
	"gkfeed/api/internal/config"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type AuthHandler struct {
	cfg             config.Config
	webAuthnService *authsvc.WebAuthnService
}

func NewAuthHandler(cfg config.Config, webAuthnService *authsvc.WebAuthnService) *AuthHandler {
	return &AuthHandler{cfg: cfg, webAuthnService: webAuthnService}
}

type registerBeginRequest struct {
	Name string `json:"name"`
}

// @Summary      Begin passkey registration
// @Description  Starts WebAuthn registration, returns creation options for the browser.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  registerBeginRequest  false  "Optional name for the passkey"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200
// @Failure      401
// @Failure      500
// @Router       /auth/register/begin [post]
func (h *AuthHandler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	var req registerBeginRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	creation, err := h.webAuthnService.BeginRegistration(user, req.Name)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("begin registration: %w", err))
		return
	}

	writeJSON(w, creation)
}

// @Summary      Finish passkey registration
// @Description  Completes WebAuthn registration with the attestation response from the browser.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  object  true  "WebAuthn attestation response"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200   {object}  object{id=string,name=string}
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /auth/register/finish [post]
func (h *AuthHandler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		http.Error(w, "Invalid credential creation response", http.StatusBadRequest)
		return
	}

	credential, name, err := h.webAuthnService.FinishRegistration(user, parsedResponse)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("finish registration: %w", err))
		return
	}

	writeJSON(w, struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{
		ID:   base64.RawURLEncoding.EncodeToString(credential.ID),
		Name: name,
	})
}

// @Summary      Begin passkey login
// @Description  Starts discoverable WebAuthn login, returns assertion options for the browser.
// @Tags         auth
// @Produce      json
// @Success      200
// @Failure      500
// @Router       /auth/login/begin [post]
func (h *AuthHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	assertion, err := h.webAuthnService.BeginLogin()
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("begin login: %w", err))
		return
	}

	writeJSON(w, assertion)
}

// @Summary      Finish passkey login
// @Description  Completes WebAuthn authentication, returns a JWT access/refresh token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object  true  "WebAuthn assertion response"
// @Success      200   {object}  object{access_token=string,refresh_token=string,credential_id=string}
// @Failure      400
// @Failure      500
// @Router       /auth/login/finish [post]
func (h *AuthHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		http.Error(w, "Invalid credential assertion response", http.StatusBadRequest)
		return
	}

	credential, user, err := h.webAuthnService.FinishLogin(parsedResponse)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("finish login: %w", err))
		return
	}

	accessToken, err := authsvc.GenerateAccessToken(user.ID, user.Name, h.cfg)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("generate access token: %w", err))
		return
	}

	refreshToken := models.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(h.cfg.RefreshTokenTTL),
	}
	if err := db.StoreRefreshToken(refreshToken); err != nil {
		writeInternalServerError(w, fmt.Errorf("store refresh token: %w", err))
		return
	}

	writeJSON(w, struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		CredentialID string `json:"credential_id"`
	}{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.ID,
		CredentialID: base64.RawURLEncoding.EncodeToString(credential.ID),
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// @Summary      Refresh access token
// @Description  Exchanges a valid refresh token for a new access/refresh token pair. Old refresh token is invalidated.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      refreshRequest  true  "Refresh token"
// @Success      200   {object}  object{access_token=string,refresh_token=string}
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	stored, err := db.GetRefreshToken(req.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	if time.Now().After(stored.ExpiresAt) {
		db.DeleteRefreshToken(stored.ID)
		http.Error(w, "Refresh token expired", http.StatusUnauthorized)
		return
	}

	if err := db.DeleteRefreshToken(stored.ID); err != nil {
		writeInternalServerError(w, fmt.Errorf("delete refresh token: %w", err))
		return
	}

	user, err := db.GetUserFromDBByID(stored.UserID)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("get user: %w", err))
		return
	}

	accessToken, err := authsvc.GenerateAccessToken(user.ID, user.Name, h.cfg)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("generate access token: %w", err))
		return
	}

	newRefreshToken := models.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(h.cfg.RefreshTokenTTL),
	}
	if err := db.StoreRefreshToken(newRefreshToken); err != nil {
		writeInternalServerError(w, fmt.Errorf("store refresh token: %w", err))
		return
	}

	writeJSON(w, struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken.ID,
	})
}

// @Summary      Logout
// @Description  Invalidates all refresh tokens for the authenticated user.
// @Tags         auth
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      204
// @Failure      401
// @Failure      500
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	if err := db.DeleteUserRefreshTokens(user.ID); err != nil {
		writeInternalServerError(w, fmt.Errorf("delete refresh tokens: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary      List passkeys
// @Description  Returns all registered passkeys for the authenticated user.
// @Tags         auth
// @Produce      json
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200  {array}   object{id=string,name=string,created_at=string,last_used_at=string}
// @Failure      401
// @Failure      500
// @Router       /auth/credentials [get]
func (h *AuthHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	infos, err := db.ListUserWebAuthnCredentials(user.ID)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("list credentials: %w", err))
		return
	}

	type credentialResponse struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		CreatedAt  string  `json:"created_at"`
		LastUsedAt *string `json:"last_used_at"`
	}

	result := make([]credentialResponse, len(infos))
	for i, info := range infos {
		result[i] = credentialResponse{
			ID:         base64.RawURLEncoding.EncodeToString(info.ID),
			Name:       info.Name,
			CreatedAt:  info.CreatedAt,
			LastUsedAt: info.LastUsedAt,
		}
	}

	writeJSON(w, result)
}

// @Summary      Delete passkey
// @Description  Removes a specific passkey credential for the authenticated user.
// @Tags         auth
// @Param        id   path      string  true  "Credential ID (base64url)"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      204
// @Failure      400
// @Failure      401
// @Failure      404
// @Failure      500
// @Router       /auth/credentials/{id} [delete]
func (h *AuthHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	credIDB64 := mux.Vars(r)["id"]
	credID, err := base64.RawURLEncoding.DecodeString(credIDB64)
	if err != nil {
		http.Error(w, "Invalid credential ID", http.StatusBadRequest)
		return
	}

	deleted, err := db.DeleteWebAuthnCredential(credID, user.ID)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("delete credential: %w", err))
		return
	}
	if !deleted {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary      Current user
// @Description  Returns the authenticated user's ID and name. JWT only.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  object{id=int,name=string}
// @Failure      401
// @Router       /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	writeJSON(w, struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   user.ID,
		Name: user.Name,
	})
}
