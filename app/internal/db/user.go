package db

import (
	"fmt"

	"gkfeed/api/internal/models"
)

func GetUserFromDB(name string) (models.User, error) {
	database, err := getDB()
	if err != nil {
		return models.User{}, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	var user models.User
	err = database.QueryRow(
		"SELECT id, name, password FROM users WHERE name = ?",
		name,
	).Scan(&user.ID, &user.Name, &user.HashedPassword)
	if err != nil {
		return models.User{}, fmt.Errorf("get user %q: %w", name, err)
	}

	return user, nil
}
