package services

import "testing"

func TestCreateFeedFromURL(t *testing.T) {
	tests := []struct {
		name      string
		inputURL  string
		wantTitle string
		wantType  string
		wantURL   string
	}{
		{
			name:      "YouTube channel",
			inputURL:  "https://www.youtube.com/@example/videos",
			wantTitle: "example",
			wantType:  feedTypeYouTube,
			wantURL:   "https://www.youtube.com/@example/videos",
		},
		{
			name:      "TikTok mirror",
			inputURL:  "https://tok.adminforge.de/@example",
			wantTitle: "example",
			wantType:  feedTypeTikTok,
			wantURL:   "https://www.tiktok.com/@example",
		},
		{
			name:      "Rezka film",
			inputURL:  "https://hdrezka.me/films/drama/example.html",
			wantTitle: "example",
			wantType:  feedTypeRezka,
			wantURL:   "https://hdrezka.me/films/drama/example.html",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			feed, err := CreateFeedFromURL(test.inputURL)
			if err != nil {
				t.Fatalf("CreateFeedFromURL() returned an error: %v", err)
			}
			if feed.Title != test.wantTitle || feed.Type != test.wantType || feed.URL != test.wantURL {
				t.Fatalf("CreateFeedFromURL() = %#v, want title %q, type %q, URL %q", feed, test.wantTitle, test.wantType, test.wantURL)
			}
		})
	}
}

func TestCreateFeedFromURLRejectsIncompleteURL(t *testing.T) {
	_, err := CreateFeedFromURL("https://www.youtube.com/@")
	if err == nil {
		t.Fatal("CreateFeedFromURL() accepted an incomplete URL")
	}
}
