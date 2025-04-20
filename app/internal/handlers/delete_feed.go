package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"net/http"
	"strconv"
)

func HandleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	var id int
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := db.GetUserFromDB(userName)
	feed := db.GetFeedByID(id)
	if feed.UserID != user.ID {
		return
	}
	db.DeleteFeedByID(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(
		struct {
			Deleted bool        `json:"deleted"`
			Item    models.Feed `json:"item"`
		}{
			true, feed,
		},
	)
}
