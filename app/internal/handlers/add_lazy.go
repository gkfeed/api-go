package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"gkfeed/api/internal/services"
	"net/http"
)

func HandleAddFeedLazy(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	var input struct {
		Url string `json:"url"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	feedFactory := services.FeedFactory{}
	feedInput, err := feedFactory.CreateFromUrl(input.Url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := db.GetUserFromDB(userName)
	feed := db.AddFeed(*feedInput, user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(
		struct {
			Created bool        `json:"created"`
			Item    models.Feed `json:"item"`
		}{
			true, feed,
		},
	)
}
