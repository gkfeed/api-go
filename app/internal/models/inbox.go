package models

import "time"

const (
	DeliveryUnread   = "unread"
	DeliveryRead     = "read"
	DeliveryArchived = "archived"
)

type Delivery struct {
	ID              string     `json:"delivery_id"`
	SenderUserID    int        `json:"sender_user_id"`
	RecipientUserID int        `json:"recipient_user_id"`
	ClonedItemID    int        `json:"item_id"`
	FeedID          int        `json:"feed_id"`
	Note            *string    `json:"note,omitempty"`
	State           string     `json:"state"`
	CreatedAt       time.Time  `json:"created_at"`
	ReadAt          *time.Time `json:"read_at,omitempty"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
}

type InboxItem struct {
	Item     Item     `json:"item"`
	Delivery Delivery `json:"delivery"`
}
