package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"gkfeed/api/internal/db"
)

type shareItemRequest struct {
	ItemID          int     `json:"item_id"`
	RecipientUserID int     `json:"recipient_user_id"`
	Note            *string `json:"note"`
}

// @Summary      Share an item into another user's Inbox feed
// @Description  Atomically creates an independent item clone and delivery. The clone is read through the normal feed APIs. Network retries must reuse Idempotency-Key.
// @Tags         feeds
// @Accept       json
// @Produce      json
// @Param        Idempotency-Key header string true "Client-generated retry key"
// @Param        share body shareItemRequest true "Share request"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200  {object} models.Delivery
// @Success      201  {object} models.Delivery
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      409
// @Failure      422
// @Failure      500
// @Router       /api/v1/shares [post]
func HandleShareItem(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 200 {
		http.Error(w, "Idempotency-Key must contain 1 to 200 characters", http.StatusBadRequest)
		return
	}

	var input shareItemRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ItemID <= 0 || input.RecipientUserID <= 0 {
		http.Error(w, "item_id and recipient_user_id must be positive", http.StatusUnprocessableEntity)
		return
	}
	if input.Note != nil && utf8.RuneCountInString(*input.Note) > 500 {
		http.Error(w, "note must not exceed 500 characters", http.StatusUnprocessableEntity)
		return
	}

	delivery, created, err := db.ShareItem(user.ID, input.ItemID, input.RecipientUserID, input.Note, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrSelfShare):
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		case errors.Is(err, db.ErrIdempotencyConflict):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, sql.ErrNoRows):
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		default:
			writeInternalServerError(w, err)
		}
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSONStatus(w, status, delivery)
}
