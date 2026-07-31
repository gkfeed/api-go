package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gkfeed/api/internal/auth"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

const (
	defaultAccessTokenTTL  = 30 * time.Minute
	defaultRefreshTokenTTL = 90 * 24 * time.Hour
)

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
}

func (h *AuthHandler) issueTokenPair(user models.User) (tokenResponse, error) {
	accessTTL := h.cfg.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = defaultAccessTokenTTL
	}
	refreshTTL := h.cfg.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTokenTTL
	}

	refreshToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return tokenResponse{}, err
	}
	familyID, err := auth.GenerateTokenFamily()
	if err != nil {
		return tokenResponse{}, err
	}
	if err := db.CreateAuthRefreshToken(
		user.ID,
		auth.DigestToken(refreshToken),
		familyID,
		time.Now().Add(refreshTTL),
	); err != nil {
		return tokenResponse{}, fmt.Errorf("store refresh token: %w", err)
	}

	if h.sessions == nil {
		h.sessions = auth.NewSessionStore(accessTTL)
	}
	accessToken, err := h.sessions.Create(models.User{ID: user.ID, Name: user.Name}, familyID)
	if err != nil {
		_, _ = db.RevokeAuthRefreshToken(auth.DigestToken(refreshToken), time.Now())
		return tokenResponse{}, fmt.Errorf("create access token: %w", err)
	}

	return tokenResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(accessTTL / time.Second),
		RefreshExpiresIn: int64(refreshTTL / time.Second),
	}, nil
}

// @Summary      Refresh access token
// @Description  Rotates a refresh token and returns a new access/refresh token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      refreshRequest  true  "Refresh token"
// @Success      200   {object}  tokenResponse
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	var req refreshRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "Refresh token is required", http.StatusBadRequest)
		return
	}

	refreshTTL := h.cfg.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTokenTTL
	}
	newRefreshToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("generate refresh token: %w", err))
		return
	}
	now := time.Now()
	rotation, err := db.RotateAuthRefreshToken(
		auth.DigestToken(req.RefreshToken),
		auth.DigestToken(newRefreshToken),
		now,
		now.Add(refreshTTL),
	)
	if err != nil {
		if errors.Is(err, db.ErrInvalidRefreshToken) || errors.Is(err, db.ErrRefreshTokenReuse) {
			if len(rotation.FamilyID) > 0 {
				h.sessions.RevokeFamily(rotation.FamilyID)
			}
			writeBearerUnauthorized(w)
			return
		}
		writeInternalServerError(w, fmt.Errorf("rotate refresh token: %w", err))
		return
	}

	accessTTL := h.cfg.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = defaultAccessTokenTTL
	}
	if h.sessions == nil {
		h.sessions = auth.NewSessionStore(accessTTL)
	}
	accessToken, err := h.sessions.Create(rotation.User, rotation.FamilyID)
	if err != nil {
		writeInternalServerError(w, fmt.Errorf("create access token: %w", err))
		return
	}

	writeJSON(w, tokenResponse{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(accessTTL / time.Second),
		RefreshExpiresIn: int64(refreshTTL / time.Second),
	})
}

// @Summary      Logout
// @Description  Revokes the refresh-token family associated with the supplied token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      refreshRequest  true  "Refresh token"
// @Success      204
// @Failure      400
// @Failure      500
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	var req refreshRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "Refresh token is required", http.StatusBadRequest)
		return
	}

	familyID, err := db.RevokeAuthRefreshToken(auth.DigestToken(req.RefreshToken), time.Now())
	if err != nil {
		if errors.Is(err, db.ErrInvalidRefreshToken) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeInternalServerError(w, fmt.Errorf("revoke refresh token: %w", err))
		return
	}
	h.sessions.RevokeFamily(familyID)
	w.WriteHeader(http.StatusNoContent)
}

// @Summary      Logout all sessions
// @Description  Revokes every refresh token and access session for the authenticated user.
// @Tags         auth
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      204
// @Failure      401
// @Failure      500
// @Router       /api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	if err := db.RevokeUserAuthRefreshTokens(user.ID, time.Now()); err != nil {
		writeInternalServerError(w, fmt.Errorf("revoke user refresh tokens: %w", err))
		return
	}
	h.sessions.RevokeUser(user.ID)
	w.WriteHeader(http.StatusNoContent)
}

func writeBearerUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
