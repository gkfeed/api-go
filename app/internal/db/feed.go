package db

import (
	"fmt"

	"gkfeed/api/internal/models"
)

const feedColumns = "feed.id, feed.title, feed.url, feed.type, feed.user_id"

func GetUserFeeds(userID int) ([]models.Feed, error) {
	return getFeeds("SELECT "+feedColumns+" FROM feed WHERE user_id = $1", userID)
}

func getFeeds(query string, args ...any) ([]models.Feed, error) {
	database, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
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
	database, err := getDB()
	if err != nil {
		return models.Feed{}, fmt.Errorf("open database: %w", err)
	}
	var feedID int
	err = database.QueryRow(
		"INSERT INTO feed (title, type, url, user_id) VALUES ($1, $2, $3, $4) RETURNING id",
		feedInput.Title,
		feedInput.Type,
		feedInput.URL,
		userID,
	).Scan(&feedID)
	if err != nil {
		return models.Feed{}, fmt.Errorf("insert feed: %w", err)
	}

	return models.Feed{
		ID:     feedID,
		Title:  feedInput.Title,
		Type:   feedInput.Type,
		URL:    feedInput.URL,
		UserID: userID,
	}, nil
}

func DeleteFeedByID(id int) error {
	database, err := getDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	_, err = database.Exec("DELETE FROM feed WHERE id = $1", id)
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
	feed, err := scanFeed(database.QueryRow("SELECT "+feedColumns+" FROM feed WHERE id = $1", id))
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
