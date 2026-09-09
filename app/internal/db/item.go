package db

import (
	"database/sql"
	"fmt"

	"gkfeed/api/internal/models"
)

const (
	itemColumns         = "item.id, item.feed_id, item.title, item.text, item.date, item.link"
	itemWithFeedColumns = itemColumns + ", " + feedColumns
	itemDeliveryColumns = "inbox_deliveries.id, inbox_deliveries.sender_user_id, inbox_deliveries.note, inbox_deliveries.created_at"
	userItemQuery       = `SELECT ` + itemWithFeedColumns + `, ` + itemDeliveryColumns + `
		FROM item
		JOIN feed ON item.feed_id = feed.id
		LEFT JOIN inbox_deliveries ON inbox_deliveries.cloned_item_id = item.id
		WHERE item.id = ? AND feed.user_id = ?`
)

func GetUserItems(userID int) ([]models.Item, error) {
	return getItems(
		`SELECT `+itemColumns+`
		FROM item
		JOIN feed ON item.feed_id = feed.id
		WHERE feed.user_id = ?
		  AND item.id NOT IN (
			SELECT item_id FROM deleted_items WHERE user_id = ?
		)`,
		userID,
		userID,
	)
}

func GetUserItemsPage(userID int, cursor *int, limit int) ([]models.Item, error) {
	return GetUserItemsPageForFeed(userID, nil, cursor, limit)
}

func GetUserItemsPageForFeed(userID int, feedID, cursor *int, limit int) ([]models.Item, error) {
	query := `SELECT ` + itemColumns + `, ` + itemDeliveryColumns + `
		FROM item
		JOIN feed ON item.feed_id = feed.id
		LEFT JOIN inbox_deliveries ON inbox_deliveries.cloned_item_id = item.id
		WHERE feed.user_id = ?
		  AND item.id NOT IN (
			SELECT item_id FROM deleted_items WHERE user_id = ?
		  )`
	args := []any{userID, userID}
	if feedID != nil {
		query += " AND feed.id = ?"
		args = append(args, *feedID)
	}

	if cursor != nil {
		query += " AND item.id < ?"
		args = append(args, *cursor)
	}

	query += " ORDER BY item.id DESC LIMIT ?"
	args = append(args, limit)
	return getItemsWithDelivery(query, args...)
}

func InsertItemsIntoDeletedItems(userID int, itemIDs []int) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	transaction, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin deleted-items transaction: %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare("INSERT INTO deleted_items (user_id, item_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("prepare deleted-item insert: %w", err)
	}
	defer statement.Close()

	for _, id := range itemIDs {
		if _, err := statement.Exec(userID, id); err != nil {
			return fmt.Errorf("insert deleted item %d: %w", id, err)
		}
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit deleted items: %w", err)
	}
	return nil
}

func GetUserItemByID(userID, itemID int) (models.Item, models.Feed, error) {
	database, err := getDB()
	if err != nil {
		return models.Item{}, models.Feed{}, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	item, feed, err := scanItemWithFeed(database.QueryRow(userItemQuery, itemID, userID))
	if err != nil {
		return models.Item{}, models.Feed{}, fmt.Errorf("get item %d for user %d: %w", itemID, userID, err)
	}
	return item, feed, nil
}

func scanItemWithFeed(row rowScanner) (models.Item, models.Feed, error) {
	var item models.Item
	var feed models.Feed
	var deliveryID, note sql.NullString
	var senderUserID sql.NullInt64
	var deliveryCreatedAt sql.NullTime
	err := row.Scan(
		&item.ID,
		&item.FeedID,
		&item.Title,
		&item.Text,
		&item.Date,
		&item.Link,
		&feed.ID,
		&feed.Title,
		&feed.URL,
		&feed.Type,
		&feed.UserID,
		&deliveryID,
		&senderUserID,
		&note,
		&deliveryCreatedAt,
	)
	setItemDelivery(&item, deliveryID, senderUserID, note, deliveryCreatedAt)
	return item, feed, err
}

func getItemsWithDelivery(query string, args ...any) ([]models.Item, error) {
	database, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()
	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	items := make([]models.Item, 0)
	for rows.Next() {
		item, err := scanItemWithDelivery(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}
	return items, nil
}

func scanItemWithDelivery(row rowScanner) (models.Item, error) {
	var item models.Item
	var deliveryID, note sql.NullString
	var senderUserID sql.NullInt64
	var deliveryCreatedAt sql.NullTime
	err := row.Scan(
		&item.ID, &item.FeedID, &item.Title, &item.Text, &item.Date, &item.Link,
		&deliveryID, &senderUserID, &note, &deliveryCreatedAt,
	)
	setItemDelivery(&item, deliveryID, senderUserID, note, deliveryCreatedAt)
	return item, err
}

func setItemDelivery(item *models.Item, id sql.NullString, senderID sql.NullInt64, note sql.NullString, createdAt sql.NullTime) {
	if !id.Valid {
		return
	}
	delivery := &models.ItemDelivery{ID: id.String, SenderUserID: int(senderID.Int64), CreatedAt: createdAt.Time}
	if note.Valid {
		delivery.Note = &note.String
	}
	item.Delivery = delivery
}

func getItems(query string, args ...any) ([]models.Item, error) {
	database, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}

	return items, nil
}

func scanItem(row rowScanner) (models.Item, error) {
	var item models.Item
	err := row.Scan(&item.ID, &item.FeedID, &item.Title, &item.Text, &item.Date, &item.Link)
	return item, err
}
