package db

import (
	"fmt"

	"gkfeed/api/internal/models"
)

const (
	itemColumns         = "item.id, item.feed_id, item.title, item.text, item.date, item.link"
	itemWithFeedColumns = itemColumns + ", " + feedColumns
	userItemQuery       = `SELECT ` + itemWithFeedColumns + `
		FROM item
		JOIN feed ON item.feed_id = feed.id
		WHERE item.id = $1 AND feed.user_id = $2`
)

func GetUserItems(userID int) ([]models.Item, error) {
	return getItems(
		`SELECT `+itemColumns+`
		FROM item
		JOIN feed ON item.feed_id = feed.id
		WHERE feed.user_id = $1
		  AND item.id NOT IN (
			SELECT item_id FROM deleted_items WHERE user_id = $2
		)`,
		userID,
		userID,
	)
}

func GetUserItemsPage(userID int, cursor *int, limit int) ([]models.Item, error) {
	query := `SELECT ` + itemColumns + `
		FROM item
		JOIN feed ON item.feed_id = feed.id
		WHERE feed.user_id = $1
		  AND item.id NOT IN (
			SELECT item_id FROM deleted_items WHERE user_id = $2
		  )`
	args := []any{userID, userID}

	if cursor != nil {
		query += fmt.Sprintf(" AND item.id < $%d", len(args)+1)
		args = append(args, *cursor)
	}

	query += fmt.Sprintf(" ORDER BY item.id DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)
	return getItems(query, args...)
}

func InsertItemsIntoDeletedItems(userID int, itemIDs []int) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	transaction, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin deleted-items transaction: %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare("INSERT INTO deleted_items (user_id, item_id) VALUES ($1, $2) ON CONFLICT DO NOTHING")
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
	item, feed, err := scanItemWithFeed(database.QueryRow(userItemQuery, itemID, userID))
	if err != nil {
		return models.Item{}, models.Feed{}, fmt.Errorf("get item %d for user %d: %w", itemID, userID, err)
	}
	return item, feed, nil
}

func scanItemWithFeed(row rowScanner) (models.Item, models.Feed, error) {
	var item models.Item
	var feed models.Feed
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
	)
	return item, feed, err
}

func getItems(query string, args ...any) ([]models.Item, error) {
	database, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
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
