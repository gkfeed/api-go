package models

type Feed struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Url    string `json:"url"`
	UserID int    `json:"userid"`
}
