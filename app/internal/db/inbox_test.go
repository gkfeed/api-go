package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"gkfeed/api/internal/models"
)

func TestInboxDeliveryLifecycle(t *testing.T) {
	useTestDatabase(t)
	if err := InitCoreSchema(); err != nil {
		t.Fatalf("InitCoreSchema() returned error: %v", err)
	}

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	_, err = database.Exec(`
		INSERT INTO users (id, name, hashed_password) VALUES (2, 'recipient', 'secret');
		INSERT INTO feed (id, title, url, type, user_id) VALUES (10, 'Source', 'https://example.com/feed', 'web', 1);
		INSERT INTO item (id, feed_id, title, text, date, link)
		VALUES (20, 10, 'Original title', 'Original text', ?, 'https://example.com/item')`,
		time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC),
	)
	database.Close()
	if err != nil {
		t.Fatalf("insert inbox fixtures: %v", err)
	}

	if _, _, err := ShareItem(1, 20, 2, nil, "before-inbox"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ShareItem() without recipient Inbox error = %v, want sql.ErrNoRows", err)
	}
	if _, err := GetInboxItems(2, ""); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetInboxItems() before explicit creation error = %v, want sql.ErrNoRows", err)
	}

	inbox, created, err := CreateInbox(2)
	if err != nil {
		t.Fatalf("CreateInbox() returned error: %v", err)
	}
	if !created || inbox.Type != "inbox" || inbox.UserID != 2 {
		t.Fatalf("CreateInbox() = (%#v, %t), want a newly-created Inbox for user 2", inbox, created)
	}
	sameInbox, created, err := CreateInbox(2)
	if err != nil {
		t.Fatalf("second CreateInbox() returned error: %v", err)
	}
	if created || sameInbox.ID != inbox.ID {
		t.Fatalf("second CreateInbox() = (%#v, %t), want existing Inbox %d", sameInbox, created, inbox.ID)
	}

	note := "Worth reading"
	delivery, created, err := ShareItem(1, 20, 2, &note, "share-1")
	if err != nil {
		t.Fatalf("ShareItem() returned error: %v", err)
	}
	if !created || delivery.ID == "" || delivery.ClonedItemID == 20 || delivery.FeedID != inbox.ID || delivery.State != models.DeliveryUnread {
		t.Fatalf("ShareItem() = (%#v, %t), want a new unread independent clone", delivery, created)
	}

	retry, created, err := ShareItem(1, 20, 2, &note, "share-1")
	if err != nil {
		t.Fatalf("ShareItem() retry returned error: %v", err)
	}
	if created || retry.ID != delivery.ID || retry.ClonedItemID != delivery.ClonedItemID {
		t.Fatalf("ShareItem() retry = (%#v, %t), want existing delivery %#v", retry, created, delivery)
	}
	differentNote := "Different payload"
	if _, _, err := ShareItem(1, 20, 2, &differentNote, "share-1"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("ShareItem() conflicting retry error = %v, want ErrIdempotencyConflict", err)
	}
	if _, _, err := ShareItem(1, 20, 1, nil, "self-share"); !errors.Is(err, ErrSelfShare) {
		t.Fatalf("ShareItem() to self error = %v, want ErrSelfShare", err)
	}

	database, err = getDB()
	if err != nil {
		t.Fatalf("reopen test database: %v", err)
	}
	if _, err := database.Exec("UPDATE item SET title = 'Changed' WHERE id = 20"); err != nil {
		database.Close()
		t.Fatalf("change source item: %v", err)
	}
	database.Close()

	items, err := GetInboxItems(2, models.DeliveryUnread)
	if err != nil {
		t.Fatalf("GetInboxItems() returned error: %v", err)
	}
	if len(items) != 1 || items[0].Item.Title != "Original title" || items[0].Delivery.ID != delivery.ID {
		t.Fatalf("GetInboxItems() = %#v, want the unchanged independent clone", items)
	}
	if filtered, err := GetInboxItems(2, models.DeliveryArchived); err != nil || len(filtered) != 0 {
		t.Fatalf("GetInboxItems(archived) = (%#v, %v), want empty result", filtered, err)
	}
	if _, err := GetInboxItems(2, "unknown"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("GetInboxItems(unknown) error = %v, want ErrInvalidState", err)
	}

	read, err := SetInboxItemState(2, delivery.ClonedItemID, models.DeliveryRead)
	if err != nil || read.State != models.DeliveryRead || read.ReadAt == nil {
		t.Fatalf("SetInboxItemState(read) = (%#v, %v), want read delivery with timestamp", read, err)
	}
	archived, err := SetInboxItemState(2, delivery.ClonedItemID, models.DeliveryArchived)
	if err != nil || archived.State != models.DeliveryArchived || archived.ArchivedAt == nil {
		t.Fatalf("SetInboxItemState(archived) = (%#v, %v), want archived delivery with timestamp", archived, err)
	}
	if _, err := SetInboxItemState(2, delivery.ClonedItemID, models.DeliveryRead); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("SetInboxItemState(read archived) error = %v, want ErrInvalidTransition", err)
	}
	if _, err := SetInboxItemState(1, delivery.ClonedItemID, models.DeliveryArchived); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("SetInboxItemState() by sender error = %v, want sql.ErrNoRows", err)
	}
}

func TestInboxDeliveryDoesNotStoreSourceItemID(t *testing.T) {
	useTestDatabase(t)
	if err := InitCoreSchema(); err != nil {
		t.Fatalf("InitCoreSchema() returned error: %v", err)
	}

	database, err := getDB()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer database.Close()
	rows, err := database.Query("PRAGMA table_info(inbox_deliveries)")
	if err != nil {
		t.Fatalf("read delivery schema: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var position, notNull, primaryKey int
		var name, dataType string
		var defaultValue any
		if err := rows.Scan(&position, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan delivery column: %v", err)
		}
		if name == "source_item_id" {
			t.Fatal("inbox_deliveries unexpectedly stores source_item_id")
		}
	}
}
