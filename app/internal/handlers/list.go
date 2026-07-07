package handlers

import (
	"encoding/json"
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
