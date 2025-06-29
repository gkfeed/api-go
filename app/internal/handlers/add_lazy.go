package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"gkfeed/api/internal/services"
	"net/http"
)

var (
	dbGetUserFromDB       = db.GetUserFromDB
	dbAddFeed             = db.AddFeed
	servicesCreateFromUrl = (&services.FeedFactory{}).CreateFromUrl
)

func HandleAddFeedLazy(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		if _, err := w.Write([]byte("No authentication provided")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
		return
	}

	var input struct {
		Url string `json:"url"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	feedInput, err := servicesCreateFromUrl(input.Url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := dbGetUserFromDB(userName)
	feed := dbAddFeed(*feedInput, user.ID)

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
