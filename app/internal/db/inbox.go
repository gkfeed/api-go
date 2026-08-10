package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"gkfeed/api/internal/models"
)

var (
	ErrIdempotencyConflict = errors.New("idempotency key was already used for a different request")
	ErrSelfShare           = errors.New("an item cannot be shared with its owner")
)

const deliveryColumns = `inbox_deliveries.id, inbox_deliveries.sender_user_id,
	inbox_deliveries.recipient_user_id, inbox_deliveries.cloned_item_id, item.feed_id,
	inbox_deliveries.note, inbox_deliveries.created_at`

func ShareItem(senderUserID, sourceItemID, recipientUserID int, note *string, idempotencyKey string) (models.Delivery, bool, error) {
	if senderUserID == recipientUserID {
		return models.Delivery{}, false, ErrSelfShare
	}
	requestHash, err := shareRequestHash(sourceItemID, recipientUserID, note)
	if err != nil {
		return models.Delivery{}, false, err
	}

	database, err := getDB()
	if err != nil {
		return models.Delivery{}, false, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	transaction, err := database.Begin()
	if err != nil {
		return models.Delivery{}, false, fmt.Errorf("begin share transaction: %w", err)
	}
	defer transaction.Rollback()

	existing, existingHash, err := getDeliveryByKey(transaction, senderUserID, idempotencyKey)
	if err == nil {
		if existingHash != requestHash {
			return models.Delivery{}, false, ErrIdempotencyConflict
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.Delivery{}, false, fmt.Errorf("check idempotency key: %w", err)
	}

	item, err := scanItem(transaction.QueryRow(
		`SELECT `+itemColumns+` FROM item
		 JOIN feed ON item.feed_id = feed.id
		 WHERE item.id = ? AND feed.user_id = ?`,
		sourceItemID, senderUserID,
	))
	if err != nil {
		return models.Delivery{}, false, fmt.Errorf("get source item: %w", err)
	}

	var inboxFeedID int
	if err := transaction.QueryRow(
		"SELECT id FROM feed WHERE user_id = ? AND type = 'inbox'", recipientUserID,
	).Scan(&inboxFeedID); err != nil {
		return models.Delivery{}, false, fmt.Errorf("get recipient inbox: %w", err)
	}

	result, err := transaction.Exec(
		"INSERT INTO item (feed_id, title, text, date, link) VALUES (?, ?, ?, ?, ?)",
		inboxFeedID, item.Title, item.Text, item.Date, item.Link,
	)
	if err != nil {
		return models.Delivery{}, false, fmt.Errorf("clone item: %w", err)
	}
	clonedItemID, err := result.LastInsertId()
	if err != nil {
		return models.Delivery{}, false, fmt.Errorf("get cloned item ID: %w", err)
	}

	deliveryID := uuid.NewString()
	_, err = transaction.Exec(
		`INSERT INTO inbox_deliveries
		 (id, sender_user_id, recipient_user_id, cloned_item_id, note, idempotency_key, request_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		deliveryID, senderUserID, recipientUserID, clonedItemID, note, idempotencyKey, requestHash,
	)
	if err != nil {
		return models.Delivery{}, false, fmt.Errorf("create delivery: %w", err)
	}

	delivery, err := scanDelivery(transaction.QueryRow(
		`SELECT `+deliveryColumns+` FROM inbox_deliveries
		 JOIN item ON item.id = inbox_deliveries.cloned_item_id
		 WHERE inbox_deliveries.id = ?`, deliveryID,
	))
	if err != nil {
		return models.Delivery{}, false, fmt.Errorf("get created delivery: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return models.Delivery{}, false, fmt.Errorf("commit share: %w", err)
	}
	return delivery, true, nil
}

func getDeliveryByKey(transaction *sql.Tx, senderUserID int, key string) (models.Delivery, string, error) {
	var delivery models.Delivery
	var requestHash string
	err := transaction.QueryRow(
		`SELECT `+deliveryColumns+`, inbox_deliveries.request_hash
		 FROM inbox_deliveries JOIN item ON item.id = inbox_deliveries.cloned_item_id
		 WHERE inbox_deliveries.sender_user_id = ? AND inbox_deliveries.idempotency_key = ?`,
		senderUserID, key,
	).Scan(
		&delivery.ID, &delivery.SenderUserID, &delivery.RecipientUserID,
		&delivery.ClonedItemID, &delivery.FeedID, &delivery.Note,
		&delivery.CreatedAt, &requestHash,
	)
	return delivery, requestHash, err
}

func scanDelivery(row rowScanner) (models.Delivery, error) {
	var delivery models.Delivery
	err := row.Scan(
		&delivery.ID, &delivery.SenderUserID, &delivery.RecipientUserID,
		&delivery.ClonedItemID, &delivery.FeedID, &delivery.Note, &delivery.CreatedAt,
	)
	return delivery, err
}

func shareRequestHash(itemID, recipientUserID int, note *string) (string, error) {
	payload, err := json.Marshal(struct {
		ItemID          int     `json:"item_id"`
		RecipientUserID int     `json:"recipient_user_id"`
		Note            *string `json:"note"`
	}{itemID, recipientUserID, note})
	if err != nil {
		return "", fmt.Errorf("encode share request: %w", err)
	}
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}
