package db

import (
	"database/sql"
	"fmt"
	"gkfeed/api/internal/models"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func GetAllFeeds() (feeds []models.Feed) {
	return getFeeds("SELECT * FROM feed;")
}

func GetUserFeeds(userID int) (feeds []models.Feed) {
	query := fmt.Sprintf("SELECT * FROM feed WHERE user_id = %d;", userID)
	return getFeeds(query)
}

func getFeeds(query string) (feeds []models.Feed) {
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
		var title string
		var feedType string
		var url string
		var userID int
		err = rows.Scan(&id, &title, &url, &feedType, &userID)
		if err != nil {
			log.Fatal(err)
		}

		feed := models.Feed{
			ID:     id,
			Title:  title,
			Type:   feedType,
			Url:    url,
			UserID: userID,
		}
		feeds = append(feeds, feed)
	}

	// Check for any errors during iteration
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return

}

func AddFeed(feedInput models.Feed, userID int) models.Feed {
	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "../data/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmtIns, err := db.Prepare("INSERT INTO feed (title, type, url, user_id) VALUES( ?, ?, ?, ? )")
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	// Execute the prepared statement
	result, err := stmtIns.Exec(feedInput.Title, feedInput.Type, feedInput.Url, userID)
	if err != nil {
		panic(err.Error())
	}

	feedID, _ := result.LastInsertId()
	return models.Feed{
		ID:     int(feedID),
		Title:  feedInput.Title,
		Type:   feedInput.Type,
		Url:    feedInput.Url,
		UserID: userID,
	}
}

func DeleteFeedByID(id int) {
	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "../data/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM feed WHERE id = ?", id)
	if err != nil {
		log.Fatal(err)
	}
}

func GetFeedByID(id int) models.Feed {
	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "../data/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM feed WHERE id = ?", id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		fmt.Println("No feed with this id: ")
		fmt.Println(id)
	}
	var title string
	var feedType string
	var url string
	var userID int
	err = rows.Scan(&id, &title, &url, &feedType, &userID)
	if err != nil {
		log.Fatal(err)
	}

	return models.Feed{
		ID:     id,
		Title:  title,
		Type:   feedType,
		Url:    url,
		UserID: userID,
	}
}
