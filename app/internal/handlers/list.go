package handlers

import (
	"gkfeed/api/internal/db"
	"net/http"
)

func HandleListOfFeeds(w http.ResponseWriter, r *http.Request) {
	userName, ok := basicAuthUserName(w, r)
	if !ok {
		return
	}

	user := db.GetUserFromDB(userName)
	feeds := db.GetUserFeeds(user.ID)

	respondJSON(w, feeds)
}
