package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

type userContextKey struct{}

var getUser = db.GetUserFromDB

func BasicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			user, authenticated, err := Authenticate(username, password)
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

func Authenticate(username, password string) (models.User, bool, error) {
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

func WithUser(ctx context.Context, user models.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(models.User)
	return user, ok
}
