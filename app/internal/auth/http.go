package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gkfeed/api/internal/config"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

type userContextKey struct{}

var getUser = db.GetUserFromDB

func BasicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			user, authenticated, err := authenticateWithDB(username, password)
			if err != nil {
				log.Printf("authentication failed: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if authenticated {
				handler(w, r.WithContext(WithUser(r.Context(), user)))
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

func Authenticate(cfg config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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

			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				log.Printf("auth: rejecting request, Authorization header=%q, parsed basic=%v", authHeader[:min(len(authHeader), 30)], ok)
			} else {
				log.Printf("auth: rejecting request, no Authorization header")
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
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

func authenticateWithDB(username, password string) (models.User, bool, error) {
	user, err := getUser(username)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, false, nil
	}
	if err != nil {
		return models.User{}, false, err
	}
	authenticated := subtle.ConstantTimeCompare([]byte(user.HashedPassword), []byte(password)) == 1
	return user, authenticated, nil
}

func AuthenticatePassword(username, password string) (models.User, bool, error) {
	return authenticateWithDB(username, password)
}

func WithUser(ctx context.Context, user models.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(models.User)
	return user, ok
}
