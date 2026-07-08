package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"net/http"
)

func HandleAddFeed(w http.ResponseWriter, r *http.Request) {
	userName, ok := basicAuthUserName(w, r)
	if !ok {
		return
	}

	var feedInput models.Feed
	err := json.NewDecoder(r.Body).Decode(&feedInput)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := db.GetUserFromDB(userName)
	feed := db.AddFeed(feedInput, user.ID)

	respondJSON(w, struct {
		Created bool        `json:"created"`
		Item    models.Feed `json:"item"`
	}{true, feed})
}
