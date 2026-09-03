package repositorytest

import (
	"errors"
	"testing"

	"gkfeed/api/internal/library"
)

type Fixture struct {
	Feeds       library.FeedRepository
	Items       library.ItemRepository
	InsertItems func(t *testing.T, feedID, count int) []int
}

type Factory func(t *testing.T) Fixture

// Run exercises behavior shared by every library storage adapter.
func Run(t *testing.T, factory Factory) {
	t.Helper()
	t.Run("ownership ordering pagination and mutations", func(t *testing.T) {
		fixture := factory(t)
		feeds, items := fixture.Feeds, fixture.Items
		ctx := t.Context()
		ownerFeed, err := feeds.Add(ctx, 1, library.CreateFeedInput{Title: "owner", Type: "rss", URL: "https://one"})
		if err != nil || ownerFeed.ID == 0 || ownerFeed.UserID != 1 {
			t.Fatalf("Add() = %#v, %v", ownerFeed, err)
		}
		otherFeed, err := feeds.Add(ctx, 2, library.CreateFeedInput{Title: "other"})
		if err != nil {
			t.Fatal(err)
		}
		ownerFeeds, err := feeds.List(ctx, 1)
		if err != nil || len(ownerFeeds) != 1 || ownerFeeds[0].ID != ownerFeed.ID {
			t.Fatalf("List() = %#v, %v", ownerFeeds, err)
		}
		empty, err := feeds.List(ctx, 99)
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("empty List() = %#v, %v", empty, err)
		}

		ids := fixture.InsertItems(t, ownerFeed.ID, 3)
		otherIDs := fixture.InsertItems(t, otherFeed.ID, 1)
		details, err := items.Get(ctx, 1, ids[0])
		if err != nil || details.Item.ID != ids[0] || details.Feed.ID != ownerFeed.ID {
			t.Fatalf("Get() = %#v, %v", details, err)
		}
		for _, test := range []struct{ userID, itemID int }{{2, ids[0]}, {1, 999999}} {
			if _, err := items.Get(ctx, test.userID, test.itemID); !errors.Is(err, library.ErrNotFound) {
				t.Fatalf("Get(%d, %d) error = %v", test.userID, test.itemID, err)
			}
		}
		listed, err := items.List(ctx, 1)
		if err != nil || len(listed) != 3 || listed[0].ID != ids[2] || listed[2].ID != ids[0] {
			t.Fatalf("List items = %#v, %v", listed, err)
		}
		first, err := items.ListPage(ctx, 1, nil, 2)
		if err != nil || len(first) != 2 {
			t.Fatalf("first page = %#v, %v", first, err)
		}
		cursor := first[1].ID
		second, err := items.ListPage(ctx, 1, &cursor, 2)
		if err != nil || len(second) != 1 || second[0].ID != ids[0] {
			t.Fatalf("second page = %#v, %v", second, err)
		}

		if err := items.Delete(ctx, 2, ids[0]); !errors.Is(err, library.ErrNotFound) {
			t.Fatalf("foreign Delete() error = %v", err)
		}
		if err := items.Delete(ctx, 1, ids[0]); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if err := items.Delete(ctx, 1, ids[0]); !errors.Is(err, library.ErrNotFound) {
			t.Fatalf("second Delete() error = %v", err)
		}
		if _, err := items.Get(ctx, 1, ids[0]); !errors.Is(err, library.ErrNotFound) {
			t.Fatalf("deleted item Get() error = %v", err)
		}

		if err := feeds.Delete(ctx, 1, otherFeed.ID); !errors.Is(err, library.ErrNotFound) {
			t.Fatalf("foreign feed Delete() error = %v", err)
		}
		if _, err := items.Get(ctx, 2, otherIDs[0]); err != nil {
			t.Fatalf("foreign feed changed: %v", err)
		}
		if err := feeds.Delete(ctx, 1, ownerFeed.ID); err != nil {
			t.Fatalf("feed Delete() error = %v", err)
		}
		for _, id := range ids[1:] {
			if _, err := items.Get(ctx, 1, id); !errors.Is(err, library.ErrNotFound) {
				t.Fatalf("cascade item %d error = %v", id, err)
			}
		}
	})
}
