package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

func HandleAddFeed(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	var feedInput models.Feed
	if !decodeJSON(w, r, &feedInput) {
		return
	}

	feed, err := db.AddFeed(feedInput, user.ID)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}

	writeJSON(w, feedMutationResponse{Created: true, Item: feed})
}
