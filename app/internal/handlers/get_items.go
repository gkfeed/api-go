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

	limit, cursor, ok := itemsPageParameters(w, r)
	if !ok {
		return
	}

	items, err := db.GetUserItemsPage(user.ID, cursor, limit+1)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}

	writeJSON(w, itemsPage(items, limit))
}

func itemsPageParameters(w http.ResponseWriter, r *http.Request) (int, *int, bool) {
	limit, ok := positiveQueryInt(w, r, "limit", defaultItemsLimit, false)
	if !ok {
		return 0, nil, false
	}

	cursor, ok := positiveQueryInt(w, r, "cursor", 0, true)
	if !ok {
		return 0, nil, false
	}
	if r.URL.Query().Get("cursor") == "" {
		return limit, nil, true
	}

	return limit, &cursor, true
}

func positiveQueryInt(w http.ResponseWriter, r *http.Request, name string, fallback int, allowZero bool) (int, bool) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return fallback, true
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 || (!allowZero && parsed == 0) {
		http.Error(w, "Invalid "+name, http.StatusBadRequest)
		return 0, false
	}

	return parsed, true
}

func itemsPage(items []models.Item, limit int) getItemsResponse {
	if len(items) <= limit {
		return getItemsResponse{Items: items}
	}

	nextCursor := items[limit-1].ID
	return getItemsResponse{Items: items[:limit], NextCursor: &nextCursor}
}
