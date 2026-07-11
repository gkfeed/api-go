package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
)

func HandleListOfFeeds(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	feeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}
	writeJSON(w, feeds)
}
