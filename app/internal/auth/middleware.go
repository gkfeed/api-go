package auth

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"gkfeed/api/internal/config"
	"gkfeed/api/internal/models"
)

func Authenticate(cfg config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return AuthenticateWithSessions(cfg, nil)
}

func AuthenticateWithSessions(cfg config.Config, sessions *SessionStore) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if sessions != nil {
				if session, ok := sessions.Get(bearerToken(r)); ok {
					log.Printf("auth: authenticated via access token user=%s id=%d", session.User.Name, session.User.ID)
					handler(w, r.WithContext(WithUser(r.Context(), session.User)))
					return
				}
			}

			if user, ok := tryJWT(r, cfg); ok {
				log.Printf("auth: authenticated via JWT user=%s id=%d", user.Name, user.ID)
				handler(w, r.WithContext(WithUser(r.Context(), user)))
				return
			}

			username, password, ok := r.BasicAuth()
			if ok {
				user, authenticated, err := authenticateWithDB(username, password)
				if err != nil {
					log.Printf("auth: basic auth error for user=%q: %v", username, err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				if authenticated {
					log.Printf("auth: authenticated via BasicAuth user=%s id=%d", user.Name, user.ID)
					handler(w, r.WithContext(WithUser(r.Context(), user)))
					return
				}
				log.Printf("auth: basic auth failed (wrong password) for user=%q", username)
			}

			if r.Header.Get("Authorization") != "" {
				log.Printf("auth: rejecting request, Authorization header provided, parsed basic=%v", ok)
			} else {
				log.Printf("auth: rejecting request, no Authorization header")
			}
			if sessions != nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
			} else {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			}
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

func bearerToken(r *http.Request) string {
	token, ok := parseBearerToken(r.Header.Get("Authorization"))
	if !ok {
		return ""
	}
	return token
}

func JWTAuth(cfg config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := tryJWT(r, cfg)
			if !ok {
				http.Error(w, "No authentication provided", http.StatusUnauthorized)
				return
			}
			handler(w, r.WithContext(WithUser(r.Context(), user)))
		}
	}
}

func tryJWT(r *http.Request, cfg config.Config) (models.User, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return models.User{}, false
	}

	tokenString, ok := parseBearerToken(header)
	if !ok {
		return models.User{}, false
	}

	claims, err := ValidateAccessToken(tokenString, cfg)
	if err != nil {
		return models.User{}, false
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return models.User{}, false
	}

	return models.User{ID: userID, Name: claims.Name}, true
}

func parseBearerToken(header string) (string, bool) {
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(header, "Bearer "), true
}
