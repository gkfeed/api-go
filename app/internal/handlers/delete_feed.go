package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

func HandleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	id, ok := queryID(w, r)
	if !ok {
		return
	}

	feed, err := db.GetFeedByID(id)
	if err != nil {
		writeLookupError(w, err)
		return
	}
	if feed.UserID != user.ID {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err := db.DeleteFeedByID(id); err != nil {
		writeInternalServerError(w, err)
		return
	}

	writeJSON(w, struct {
		Deleted bool        `json:"deleted"`
		Item    models.Feed `json:"item"`
	}{Deleted: true, Item: feed})
}
