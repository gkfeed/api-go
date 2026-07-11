package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

func HandleGetItemByID(w http.ResponseWriter, r *http.Request) {
	id, ok := queryID(w, r)
	if !ok {
		return
	}

	item, err := db.GetItemByID(id)
	if err != nil {
		writeLookupError(w, err)
		return
	}
	feed, err := db.GetFeedByID(item.FeedID)
	if err != nil {
		writeLookupError(w, err)
		return
	}

	writeJSON(w, struct {
		Item models.Item `json:"item"`
		Feed models.Feed `json:"feed"`
	}{Item: item, Feed: feed})
}
