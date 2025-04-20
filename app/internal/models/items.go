package models

import (
	"time"
)

type Item struct {
	ID     int       `json:"id"`
	FeedID int       `json:"feed_id"`
	Title  string    `json:"title"`
	Text   string    `json:"text"`
	Date   time.Time `json:"date"`
	Link   string    `json:"link"`
}
