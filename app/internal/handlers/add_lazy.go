package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"gkfeed/api/internal/services"
)

var (
	dbAddFeed             = db.AddFeed
	servicesCreateFromURL = services.CreateFeedFromURL
)

type feedMutationResponse struct {
	Created bool        `json:"created"`
	Item    models.Feed `json:"item"`
}

func HandleAddFeedLazy(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	var input struct {
		URL string `json:"url"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}

	feedInput, err := servicesCreateFromURL(input.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	feed, err := dbAddFeed(feedInput, user.ID)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}

	writeJSON(w, feedMutationResponse{Created: true, Item: feed})
}
