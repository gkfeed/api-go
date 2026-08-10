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
	ErrInvalidState        = errors.New("invalid delivery state")
	ErrInvalidTransition   = errors.New("delivery state transition is not allowed")
)

const deliveryColumns = `inbox_deliveries.id, inbox_deliveries.sender_user_id,
	inbox_deliveries.recipient_user_id, inbox_deliveries.cloned_item_id, item.feed_id,
	inbox_deliveries.note, inbox_deliveries.state, inbox_deliveries.created_at,
	inbox_deliveries.read_at, inbox_deliveries.archived_at`

func CreateInbox(userID int) (models.Feed, bool, error) {
	database, err := getDB()
	if err != nil {
		return models.Feed{}, false, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	result, err := database.Exec(
		"INSERT OR IGNORE INTO feed (title, type, url, user_id) VALUES ('Inbox', 'inbox', '', ?)",
		userID,
	)
	if err != nil {
		return models.Feed{}, false, fmt.Errorf("create inbox: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return models.Feed{}, false, fmt.Errorf("check inbox creation: %w", err)
	}

	feed, err := scanFeed(database.QueryRow("SELECT "+feedColumns+" FROM feed WHERE user_id = ? AND type = 'inbox'", userID))
	if err != nil {
		return models.Feed{}, false, fmt.Errorf("get inbox: %w", err)
	}
	return feed, rows == 1, nil
}

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

func GetInboxItems(recipientUserID int, state string) ([]models.InboxItem, error) {
	if state != "" && !validDeliveryState(state) {
		return nil, ErrInvalidState
	}
	database, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()
	var inboxFeedID int
	if err := database.QueryRow(
		"SELECT id FROM feed WHERE user_id = ? AND type = 'inbox'", recipientUserID,
	).Scan(&inboxFeedID); err != nil {
		return nil, fmt.Errorf("get inbox: %w", err)
	}

	query := `SELECT ` + itemColumns + `, ` + deliveryColumns + `
		FROM inbox_deliveries
		JOIN item ON item.id = inbox_deliveries.cloned_item_id
		WHERE inbox_deliveries.recipient_user_id = ?`
	args := []any{recipientUserID}
	if state != "" {
		query += " AND inbox_deliveries.state = ?"
		args = append(args, state)
	}
	query += " ORDER BY inbox_deliveries.created_at DESC, item.id DESC"

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query inbox items: %w", err)
	}
	defer rows.Close()

	items := make([]models.InboxItem, 0)
	for rows.Next() {
		var inboxItem models.InboxItem
		if err := rows.Scan(
			&inboxItem.Item.ID, &inboxItem.Item.FeedID, &inboxItem.Item.Title,
			&inboxItem.Item.Text, &inboxItem.Item.Date, &inboxItem.Item.Link,
			&inboxItem.Delivery.ID, &inboxItem.Delivery.SenderUserID,
			&inboxItem.Delivery.RecipientUserID, &inboxItem.Delivery.ClonedItemID,
			&inboxItem.Delivery.FeedID, &inboxItem.Delivery.Note, &inboxItem.Delivery.State,
			&inboxItem.Delivery.CreatedAt, &inboxItem.Delivery.ReadAt, &inboxItem.Delivery.ArchivedAt,
		); err != nil {
			return nil, fmt.Errorf("scan inbox item: %w", err)
		}
		items = append(items, inboxItem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inbox items: %w", err)
	}
	return items, nil
}

func SetInboxItemState(recipientUserID, itemID int, state string) (models.Delivery, error) {
	if state != models.DeliveryRead && state != models.DeliveryArchived {
		return models.Delivery{}, ErrInvalidState
	}
	database, err := getDB()
	if err != nil {
		return models.Delivery{}, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	timestampColumn := "read_at"
	if state == models.DeliveryArchived {
		timestampColumn = "archived_at"
	}
	query := `UPDATE inbox_deliveries SET state = ?, ` + timestampColumn + ` = COALESCE(` + timestampColumn + `, CURRENT_TIMESTAMP)
		 WHERE recipient_user_id = ? AND cloned_item_id = ?`
	if state == models.DeliveryRead {
		query += " AND state != 'archived'"
	}
	result, err := database.Exec(
		query,
		state, recipientUserID, itemID,
	)
	if err != nil {
		return models.Delivery{}, fmt.Errorf("update delivery state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return models.Delivery{}, fmt.Errorf("check delivery update: %w", err)
	}
	if rows == 0 {
		var currentState string
		err := database.QueryRow(
			"SELECT state FROM inbox_deliveries WHERE recipient_user_id = ? AND cloned_item_id = ?",
			recipientUserID, itemID,
		).Scan(&currentState)
		if err == nil {
			return models.Delivery{}, ErrInvalidTransition
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return models.Delivery{}, fmt.Errorf("get delivery state: %w", err)
		}
		return models.Delivery{}, sql.ErrNoRows
	}

	delivery, err := scanDelivery(database.QueryRow(
		`SELECT `+deliveryColumns+` FROM inbox_deliveries
		 JOIN item ON item.id = inbox_deliveries.cloned_item_id
		 WHERE inbox_deliveries.recipient_user_id = ? AND inbox_deliveries.cloned_item_id = ?`,
		recipientUserID, itemID,
	))
	if err != nil {
		return models.Delivery{}, fmt.Errorf("get updated delivery: %w", err)
	}
	return delivery, nil
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
		&delivery.ClonedItemID, &delivery.FeedID, &delivery.Note, &delivery.State,
		&delivery.CreatedAt, &delivery.ReadAt, &delivery.ArchivedAt, &requestHash,
	)
	return delivery, requestHash, err
}

func scanDelivery(row rowScanner) (models.Delivery, error) {
	var delivery models.Delivery
	err := row.Scan(
		&delivery.ID, &delivery.SenderUserID, &delivery.RecipientUserID,
		&delivery.ClonedItemID, &delivery.FeedID, &delivery.Note, &delivery.State,
		&delivery.CreatedAt, &delivery.ReadAt, &delivery.ArchivedAt,
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

func validDeliveryState(state string) bool {
	return state == models.DeliveryUnread || state == models.DeliveryRead || state == models.DeliveryArchived
}
