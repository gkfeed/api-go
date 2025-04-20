package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"net/http"
)

func HandleAddDeletedItems(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
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
