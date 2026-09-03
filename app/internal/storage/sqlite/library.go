package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gkfeed/api/internal/library"
)

const (
	feedColumns = "feed.id, feed.title, feed.url, feed.type, feed.user_id"
	itemColumns = "item.id, item.feed_id, item.title, item.text, item.date, item.link"
)

type FeedRepository struct{ db *sql.DB }

func NewFeedRepository(db *sql.DB) *FeedRepository { return &FeedRepository{db: db} }

func (r *FeedRepository) List(ctx context.Context, userID int) ([]library.Feed, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+feedColumns+" FROM feed WHERE user_id = ? ORDER BY id DESC", userID)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	feeds := make([]library.Feed, 0)
	for rows.Next() {
		feed, err := scanFeed(rows)
		if err != nil {
			return nil, fmt.Errorf("scan feed: %w", err)
		}
		feeds = append(feeds, feed)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feeds: %w", err)
	}
	return feeds, nil
}

func (r *FeedRepository) Add(ctx context.Context, userID int, input library.CreateFeedInput) (library.Feed, error) {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO feed (title, type, url, user_id) VALUES (?, ?, ?, ?)",
		input.Title, input.Type, input.URL, userID,
	)
	if err != nil {
		return library.Feed{}, fmt.Errorf("insert feed: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return library.Feed{}, fmt.Errorf("get inserted feed ID: %w", err)
	}
	return library.Feed{ID: int(id), Title: input.Title, Type: input.Type, URL: input.URL, UserID: userID}, nil
}

func (r *FeedRepository) Delete(ctx context.Context, userID, feedID int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin feed delete: %w", err)
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT 1 FROM feed WHERE id = ? AND user_id = ?", feedID, userID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.ErrNotFound
		}
		return fmt.Errorf("find feed: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM item WHERE feed_id = ?", feedID); err != nil {
		return fmt.Errorf("delete feed items: %w", err)
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM feed WHERE id = ? AND user_id = ?", feedID, userID)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted feeds: %w", err)
	}
	if affected != 1 {
		return library.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit feed delete: %w", err)
	}
	return nil
}

type ItemRepository struct{ db *sql.DB }

func NewItemRepository(db *sql.DB) *ItemRepository { return &ItemRepository{db: db} }

func (r *ItemRepository) Get(ctx context.Context, userID, itemID int) (library.ItemDetails, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+itemColumns+`, `+feedColumns+`
		FROM item JOIN feed ON item.feed_id = feed.id
		WHERE item.id = ? AND feed.user_id = ?`, itemID, userID)
	item, feed, err := scanItemWithFeed(row)
	if errors.Is(err, sql.ErrNoRows) {
		return library.ItemDetails{}, library.ErrNotFound
	}
	if err != nil {
		return library.ItemDetails{}, fmt.Errorf("get item: %w", err)
	}
	return library.ItemDetails{Item: item, Feed: feed}, nil
}

func (r *ItemRepository) List(ctx context.Context, userID int) ([]library.Item, error) {
	return r.list(ctx, `SELECT `+itemColumns+`
		FROM item JOIN feed ON item.feed_id = feed.id
		WHERE feed.user_id = ? ORDER BY item.id DESC`, userID)
}

func (r *ItemRepository) ListPage(ctx context.Context, userID int, cursor *int, limit int) ([]library.Item, error) {
	query := `SELECT ` + itemColumns + `
		FROM item JOIN feed ON item.feed_id = feed.id
		WHERE feed.user_id = ?`
	args := []any{userID}
	if cursor != nil {
		query += " AND item.id < ?"
		args = append(args, *cursor)
	}
	query += " ORDER BY item.id DESC LIMIT ?"
	args = append(args, limit)
	return r.list(ctx, query, args...)
}

func (r *ItemRepository) Delete(ctx context.Context, userID, itemID int) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM item WHERE id = ? AND EXISTS (
		SELECT 1 FROM feed WHERE feed.id = item.feed_id AND feed.user_id = ?
	)`, itemID, userID)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted items: %w", err)
	}
	if affected == 0 {
		return library.ErrNotFound
	}
	return nil
}

func (r *ItemRepository) list(ctx context.Context, query string, args ...any) ([]library.Item, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()
	items := make([]library.Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}
	return items, nil
}

type rowScanner interface{ Scan(...any) error }

func scanFeed(row rowScanner) (library.Feed, error) {
	var feed library.Feed
	err := row.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Type, &feed.UserID)
	return feed, err
}

func scanItem(row rowScanner) (library.Item, error) {
	var item library.Item
	err := row.Scan(&item.ID, &item.FeedID, &item.Title, &item.Text, &item.Date, &item.Link)
	return item, err
}

func scanItemWithFeed(row rowScanner) (library.Item, library.Feed, error) {
	var item library.Item
	var feed library.Feed
	err := row.Scan(&item.ID, &item.FeedID, &item.Title, &item.Text, &item.Date, &item.Link,
		&feed.ID, &feed.Title, &feed.URL, &feed.Type, &feed.UserID)
	return item, feed, err
}
