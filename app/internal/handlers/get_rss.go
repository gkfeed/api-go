package handlers

import (
	"encoding/xml"
	"net/http"
	"time"

	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"gkfeed/api/internal/services/rss"
	"gkfeed/api/pkg/auth"
)

// NOTE: deprecated
func HandleGetRSSFeed(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	user, authenticated, err := auth.Authenticate(username, password)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}
	if !authenticated {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	responseWithRSSFeed(w, user)
}

func HandleRSSFeed(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	responseWithRSSFeed(w, user)
}

func responseWithRSSFeed(w http.ResponseWriter, user models.User) {
	items, err := db.GetUserItems(user.ID)
	if err != nil {
		writeInternalServerError(w, err)
		return
	}

	rssItems := make([]rss.Item, 0, len(items))
	for _, item := range items {
		rssItems = append(rssItems, rss.Item{
			ID:          item.ID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Text,
			PubDate:     item.Date.Format(time.RFC1123),
		})
	}
	rssFeed := rss.GenerateRSS(rssItems)

	w.Header().Set("Content-Type", "application/rss+xml")
	if err := xml.NewEncoder(w).Encode(rssFeed); err != nil {
		http.Error(w, "Failed to encode RSS feed", http.StatusInternalServerError)
		return
	}
}
