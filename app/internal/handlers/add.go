package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

// @Summary      Add feed
// @Description  Adds a new RSS/YouTube/TikTok feed for the authenticated user.
// @Tags         feeds
// @Accept       json
// @Produce      json
// @Param        feed  body      models.Feed  true  "Feed to add (title, type, url)"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200   {object}  feedMutationResponse
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /api/v1/add [post]
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
