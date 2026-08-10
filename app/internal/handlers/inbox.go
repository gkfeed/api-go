package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gorilla/mux"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
)

type shareItemRequest struct {
	ItemID          int     `json:"item_id"`
	RecipientUserID int     `json:"recipient_user_id"`
	Note            *string `json:"note"`
}

type inboxItemsResponse struct {
	Items []models.InboxItem `json:"items"`
}

// @Summary      Create the authenticated user's Inbox
// @Description  Explicitly creates one Inbox feed, or returns the existing Inbox.
// @Tags         inbox
// @Produce      json
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200  {object} models.Feed
// @Success      201  {object} models.Feed
// @Failure      401
// @Failure      500
// @Router       /api/v1/feeds/me/inbox [post]
func HandleCreateInbox(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	feed, created, err := db.CreateInbox(user.ID)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSONStatus(w, status, feed)
}

// @Summary      Share an item into another user's Inbox
// @Description  Atomically creates an independent item clone and delivery. Network retries must reuse Idempotency-Key.
// @Tags         inbox
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

// @Summary      List Inbox items
// @Description  Lists the authenticated user's independent Inbox item clones and delivery metadata.
// @Tags         inbox
// @Produce      json
// @Param        state query string false "unread, read, or archived"
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200  {object} inboxItemsResponse
// @Failure      400
// @Failure      401
// @Failure      404
// @Failure      500
// @Router       /api/v1/feeds/me/inbox/items [get]
func HandleGetInboxItems(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	items, err := db.GetInboxItems(user.ID, r.URL.Query().Get("state"))
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidState):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, sql.ErrNoRows):
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		default:
			writeInternalServerError(w, err)
		}
		return
	}
	writeJSON(w, inboxItemsResponse{Items: items})
}

// @Summary      Mark an Inbox item as read
// @Tags         inbox
// @Produce      json
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200  {object} models.Delivery
// @Failure      400
// @Failure      401
// @Failure      404
// @Failure      409
// @Router       /api/v1/inbox/items/{item_id}/read [post]
func HandleReadInboxItem(w http.ResponseWriter, r *http.Request) {
	handleInboxItemState(w, r, models.DeliveryRead)
}

// @Summary      Archive an Inbox item
// @Tags         inbox
// @Produce      json
// @Security     BasicAuth
// @Security     BearerAuth
// @Success      200  {object} models.Delivery
// @Failure      400
// @Failure      401
// @Failure      404
// @Router       /api/v1/inbox/items/{item_id}/archive [post]
func HandleArchiveInboxItem(w http.ResponseWriter, r *http.Request) {
	handleInboxItemState(w, r, models.DeliveryArchived)
}

func handleInboxItemState(w http.ResponseWriter, r *http.Request, state string) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	itemID, err := strconv.Atoi(mux.Vars(r)["item_id"])
	if err != nil || itemID <= 0 {
		http.Error(w, "Invalid item_id", http.StatusBadRequest)
		return
	}

	delivery, err := db.SetInboxItemState(user.ID, itemID, state)
	if err != nil {
		if errors.Is(err, db.ErrInvalidTransition) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		writeInternalServerError(w, err)
		return
	}
	writeJSON(w, delivery)
}
