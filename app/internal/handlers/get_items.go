package handlers

import (
	"encoding/json"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"net/http"
	"strconv"
)

const defaultItemsLimit = 100

type getItemsResponse struct {
	Items      []models.Item `json:"items"`
	NextCursor *int          `json:"next_cursor,omitempty"`
}

func HandleGetItems(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		if _, err := w.Write([]byte("No authentication provided")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
		return
	}

	user := db.GetUserFromDB(userName)
	limit := defaultItemsLimit
	if limitValue := r.URL.Query().Get("limit"); limitValue != "" {
		parsedLimit, err := strconv.Atoi(limitValue)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}

		limit = parsedLimit
	}

	var cursor *int
	if cursorValue := r.URL.Query().Get("cursor"); cursorValue != "" {
		parsedCursor, err := strconv.Atoi(cursorValue)
		if err != nil || parsedCursor <= 0 {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}

		cursor = &parsedCursor
	}

	items := db.GetUserItemsPage(user.ID, cursor, limit+1)

	var nextCursor *int
	if len(items) > limit {
		nextCursorValue := items[limit-1].ID
		nextCursor = &nextCursorValue
		items = items[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(getItemsResponse{Items: items, NextCursor: nextCursor}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
