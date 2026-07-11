package handlers

import (
	"net/http"

	"gkfeed/api/internal/db"
)

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
