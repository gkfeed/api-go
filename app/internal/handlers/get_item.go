package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

type getItemResponse struct {
	Item models.Item `json:"item"`
	Feed models.Feed `json:"feed"`
}

// @Summary      Get item by ID
// @Description  Returns a single item with its parent feed for the authenticated user.
// @Tags         items
// @Produce      json
// @Param        id   query     int  true  "Item ID"
// @Success      200  {object}  object{item=object,feed=object}
// @Failure      400
// @Failure      401
// @Failure      404
// @Security     BasicAuth
// @Security     BearerAuth
// @Router       /api/v1/item [get]
func HandleGetItemByID(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	itemID, ok := queryID(w, r)
	if !ok {
		return
	}

	item, feed, err := db.GetUserItemByID(user.ID, itemID)
	if err != nil {
		writeLookupError(w, err)
		return
	}

	writeJSON(w, getItemResponse{Item: item, Feed: feed})
}
