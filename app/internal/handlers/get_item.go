package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

// @Summary      Get item by ID
// @Description  Returns a single item with its parent feed. No authentication required.
// @Tags         items
// @Produce      json
// @Param        id   query     int  true  "Item ID"
// @Success      200  {object}  object{item=object,feed=object}
// @Failure      400
// @Failure      404
// @Router       /api/v1/item [get]
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
