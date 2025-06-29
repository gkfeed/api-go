package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"net/http"
)

func HandleAddFeed(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		if _, err := w.Write([]byte("No authentication provided")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(
		struct {
			Created bool        `json:"created"`
			Item    models.Feed `json:"item"`
		}{
			true, feed,
		},
	); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
