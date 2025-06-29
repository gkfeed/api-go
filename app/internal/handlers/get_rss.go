package handlers

import (
	"encoding/xml"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"
	"gkfeed/api/internal/services/rss"
	"net/http"
	"time"
)

// NOTE: deprecated
func HandleGetRSSFeed(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	user := db.GetUserFromDB(username)
	if user.HashedPassword != password {
		if _, err := w.Write([]byte("No authentication provided")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
		return
	}

	responseWithRSSFeed(w, user)
}

func HandleRSSFeed(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		if _, err := w.Write([]byte("No authentication provided")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
		return
	}

	user := db.GetUserFromDB(userName)
	responseWithRSSFeed(w, user)
}

func responseWithRSSFeed(w http.ResponseWriter, user models.User) {
	items := db.GetUserItems(user.ID)

	var rssItems []rss.Item
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
