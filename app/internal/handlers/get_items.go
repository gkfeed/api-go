package handlers

import (
	"net/http"
	"strconv"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

const defaultItemsLimit = 100

type getItemsResponse struct {
	Items      []models.Item `json:"items"`
	NextCursor *int          `json:"next_cursor,omitempty"`
}

func HandleGetItems(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

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
		if err != nil || parsedCursor < 0 {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}

		cursor = &parsedCursor
	}

	items, err := db.GetUserItemsPage(user.ID, cursor, limit+1)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}

	var nextCursor *int
	if len(items) > limit {
		nextCursorValue := items[limit-1].ID
		nextCursor = &nextCursorValue
		items = items[:limit]
	}

	writeJSON(w, getItemsResponse{Items: items, NextCursor: nextCursor})
}
