package models

type User struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	HashedPassword string `json:"hashedPassword"`
}
