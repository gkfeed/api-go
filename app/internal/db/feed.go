package db

import (
	"fmt"

	"gkfeed/api/internal/models"
)

const feedColumns = "feed.id, feed.title, feed.url, feed.type, feed.user_id"

func GetUserFeeds(userID int) ([]models.Feed, error) {
	return getFeeds("SELECT "+feedColumns+" FROM feed WHERE user_id = ?", userID)
}

func getFeeds(query string, args ...any) ([]models.Feed, error) {
	database, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		feed, err := scanFeed(rows)
		if err != nil {
			return nil, fmt.Errorf("scan feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feeds: %w", err)
	}

	return feeds, nil
}

func AddFeed(feedInput models.Feed, userID int) (models.Feed, error) {
	feed, _, err := AddFeedWithStatus(feedInput, userID)
	return feed, err
}

func AddFeedWithStatus(feedInput models.Feed, userID int) (models.Feed, bool, error) {
	database, err := getDB()
	if err != nil {
		return models.Feed{}, false, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	if feedInput.Type == "inbox" {
		feedInput.Title = "Inbox"
		feedInput.URL = ""
		result, err := database.Exec(
			"INSERT OR IGNORE INTO feed (title, type, url, user_id) VALUES (?, 'inbox', '', ?)",
			feedInput.Title, userID,
		)
		if err != nil {
			return models.Feed{}, false, fmt.Errorf("insert inbox feed: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return models.Feed{}, false, fmt.Errorf("check inbox feed creation: %w", err)
		}
		feed, err := scanFeed(database.QueryRow(
			"SELECT "+feedColumns+" FROM feed WHERE user_id = ? AND type = 'inbox'", userID,
		))
		if err != nil {
			return models.Feed{}, false, fmt.Errorf("get inbox feed: %w", err)
		}
		return feed, rows == 1, nil
	}

	result, err := database.Exec(
		"INSERT INTO feed (title, type, url, user_id) VALUES (?, ?, ?, ?)",
		feedInput.Title,
		feedInput.Type,
		feedInput.URL,
		userID,
	)
	if err != nil {
		return models.Feed{}, false, fmt.Errorf("insert feed: %w", err)
	}

	feedID, err := result.LastInsertId()
	if err != nil {
		return models.Feed{}, false, fmt.Errorf("get inserted feed ID: %w", err)
	}

	return models.Feed{
		ID:     int(feedID),
		Title:  feedInput.Title,
		Type:   feedInput.Type,
		URL:    feedInput.URL,
		UserID: userID,
	}, true, nil
}

func DeleteFeedByID(id int) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	_, err = database.Exec("DELETE FROM feed WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete feed %d: %w", id, err)
	}
	return nil
}

func GetFeedByID(id int) (models.Feed, error) {
	database, err := getDB()
	if err != nil {
		return models.Feed{}, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	feed, err := scanFeed(database.QueryRow("SELECT "+feedColumns+" FROM feed WHERE id = ?", id))
	if err != nil {
		return models.Feed{}, fmt.Errorf("get feed %d: %w", id, err)
	}
	return feed, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFeed(row rowScanner) (models.Feed, error) {
	var feed models.Feed
	err := row.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Type, &feed.UserID)
	return feed, err
}
