package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

// @Summary      Delete feed
// @Description  Deletes a feed by ID. Only the feed owner can delete it.
// @Tags         feeds
// @Produce      json
// @Param        id   query     int  true  "Feed ID"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200  {object}  object{deleted=bool}
// @Failure      400
// @Failure      401
// @Failure      404
// @Failure      409
// @Failure      500
// @Router       /api/v1/delete [delete]
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
	if feed.Type == "inbox" {
		http.Error(w, "Inbox feeds cannot be deleted", http.StatusConflict)
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
