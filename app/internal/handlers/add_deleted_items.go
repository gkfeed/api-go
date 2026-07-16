package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
)

// @Summary      Mark items as deleted
// @Description  Marks the specified items as deleted for the authenticated user.
// @Tags         items
// @Accept       json
// @Produce      json
// @Param        body  body      object{itemIds=[]int}  true  "Item IDs to mark as deleted"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /api/v1/add_deleted_items [post]
func HandleAddDeletedItems(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	var input struct {
		ItemIDs []int `json:"itemIds"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}

	if err := db.InsertItemsIntoDeletedItems(user.ID, input.ItemIDs); err != nil {
		writeInternalServerError(w, err)
	}
}
