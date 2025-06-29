package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"net/http"
)

func HandleAddDeletedItems(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		if _, err := w.Write([]byte("No authentication provided")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
		return
	}

	var input struct {
		ItemIDs []int `json:"itemIds"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := db.GetUserFromDB(userName)
	db.InsertItemsIntoDeletedItems(user.ID, input.ItemIDs)
}
