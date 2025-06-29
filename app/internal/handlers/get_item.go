package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"net/http"
	"strconv"
)

func HandleGetItemByID(w http.ResponseWriter, r *http.Request) {
	var id int
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	item := db.GetItemByID(id)
	feed := db.GetFeedByID(item.FeedID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS,DELETE,PUT")
	if err := json.NewEncoder(w).Encode(
		struct {
			Item models.Item `json:"item"`
			Feed models.Feed `json:"feed"`
		}{
			item, feed,
		},
	); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
