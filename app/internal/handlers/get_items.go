package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"net/http"
)

func HandleGetItems(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	user := db.GetUserFromDB(userName)
	items := db.GetUserItems(user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(
		struct {
			Items []models.Item `json:"items"`
		}{
			items,
		},
	)
}
