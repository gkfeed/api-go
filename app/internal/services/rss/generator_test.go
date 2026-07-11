package rss

import "testing"

func TestGenerateRSS(t *testing.T) {
	items := []Item{{ID: 1, Title: "Example"}}
	feed := GenerateRSS(items)

	if feed.Version != "2.0" {
		t.Fatalf("Version = %q, want 2.0", feed.Version)
	}
	if feed.Channel.Title != channelTitle || feed.Channel.Link != channelLink || feed.Channel.Description != channelDescription {
		t.Fatalf("Channel contains unexpected metadata: %#v", feed.Channel)
	}
	if len(feed.Channel.Items) != 1 || feed.Channel.Items[0].ID != 1 {
		t.Fatalf("Items = %#v, want the input item", feed.Channel.Items)
	}
}
