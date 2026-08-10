package models

import "time"

// Delivery records who shared an independent item clone. The clone itself is
// consumed through the normal feed and item APIs.
type Delivery struct {
	ID              string    `json:"delivery_id"`
	SenderUserID    int       `json:"sender_user_id"`
	RecipientUserID int       `json:"recipient_user_id"`
	ClonedItemID    int       `json:"item_id"`
	FeedID          int       `json:"feed_id"`
	Note            *string   `json:"note,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ItemDelivery is included on a regular item when it was received through a
// share. It does not change how the item is listed, fetched, or archived.
type ItemDelivery struct {
	ID           string    `json:"delivery_id"`
	SenderUserID int       `json:"sender_user_id"`
	Note         *string   `json:"note,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
