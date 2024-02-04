package db

import (
	"database/sql"
	"fmt"
	"gakawarstone/rest-api/models"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func GetUserItems(userID int) (items []models.Item) {
	query := fmt.Sprintf(
		"SELECT item.* FROM item INNER JOIN feed ON item.feed_id = feed.id where feed.user_id = %d;",
		userID,
	)

	return getItems(query)
}

func GetItemByID(id int) (item models.Item) {
	query := fmt.Sprintf("SELECT * FROM item WHERE id = %d;", id)

	return getItems(query)[0]
}

func getItems(query string) (items []models.Item) {
	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "../data/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Execute a query
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		var id int
		var feedID int
		var title string
		var text string
		var date time.Time
		var link string

		err = rows.Scan(&id, &feedID, &title, &text, &date, &link)
		if err != nil {
			log.Fatal(err)
		}

		item := models.Item{
			ID:     id,
			FeedID: feedID,
			Title:  title,
			Text:   text,
			Date:   date,
			Link:   link,
		}
		items = append(items, item)
	}

	// Check for any errors during iteration
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return
}
