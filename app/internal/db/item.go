package db

import (
	"database/sql"
	"fmt"
	"gkfeed/api/internal/models"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func GetUserItems(userID int) (items []models.Item) {
	query := fmt.Sprintf(
		"SELECT item.* FROM item INNER JOIN feed ON item.feed_id = feed.id where feed.user_id = %d AND item.id NOT IN (SELECT item_id FROM deleted_items where user_id = %d);",
		userID, userID,
	)

	return getItems(query)
}

func GetUserDeletedItemsIDs(userID int) []int {
	db, err := sql.Open("sqlite3", "../data/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var itemsIDs []int

	query := fmt.Sprintf("SELECT item_id FROM deleted_items WHERE user_id = %d;", userID)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int

		err = rows.Scan(&id)
		if err != nil {
			log.Fatal(err)
		}

		itemsIDs = append(itemsIDs, id)
	}

	return itemsIDs
}

func InsertItemsIntoDeletedItems(userID int, itemIDs []int) {
	db, err := sql.Open("sqlite3", "../data/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmtIns, err := db.Prepare("INSERT INTO deleted_items (user_id, item_id) VALUES( ?, ? )")
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	for _, id := range itemIDs {
		_, err := stmtIns.Exec(userID, id)
		if err != nil {
			panic(err.Error())
		}
	}
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
