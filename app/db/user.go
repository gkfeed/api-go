package db

import (
	"fmt"
	"gakawarstone/rest-api/models"
	"log"
)

func GetUserFromDB(name string) models.User {
	// Open a connection to the SQLite database
	db, err := getDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM users WHERE name = ?;", name)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		fmt.Println("No user with this credentials")
	}
	var id int
	var userName string
	var hashedPassword string
	err = rows.Scan(&id, &userName, &hashedPassword)
	if err != nil {
		log.Fatal(err)
	}

	return models.User{
		ID:             id,
		Name:           userName,
		HashedPassword: hashedPassword,
	}
}
