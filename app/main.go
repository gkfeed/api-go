package main

import (
	"encoding/json"
	"encoding/xml"
	"gakawarstone/rest-api/db"
	"gakawarstone/rest-api/models"
	"gakawarstone/rest-api/services"
	"gakawarstone/rest-api/services/rss"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/list", basicAuth(handleListOfFeeds)).Methods("GET")
	r.HandleFunc("/api/v1/feed", basicAuth(handleRSSFeed)).Methods("GET")
	r.HandleFunc("/api/v1/add", basicAuth(handleAddFeed)).Methods("POST")
	r.HandleFunc("/api/v1/delete", basicAuth(handleDeleteFeed)).Methods("GET")
	r.HandleFunc("/api/v1/add_lazy", basicAuth(handleAddFeedLazy)).Methods("POST")
	r.HandleFunc("/api/v1/item", handleGetItemByID).Methods("GET")

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost", "http://localhost:4200", "HTTP://localhost:8086"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(r)

	http.ListenAndServe(":8086", corsHandler)
}

func basicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			user := db.GetUserFromDB(username)
			if user.HashedPassword == password {
				handler(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

func handleListOfFeeds(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	user := db.GetUserFromDB(userName)
	feeds := db.GetUserFeeds(user.ID)

	w.Header().Set("Content-Type", "application/json")
	// w.Header().Set("Access-Control-Allow-Headers:", "*")
	// w.Header().Set("Access-Control-Allow-Origin", "*")
	// w.Header().Set("Access-Control-Allow-Methods", "*")
	json.NewEncoder(w).Encode(feeds)
}

func handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	user := db.GetUserFromDB(userName)
	items := db.GetUserItems(user.ID)

	var rssItems []rss.Item
	for _, item := range items {
		rssItems = append(rssItems, rss.Item{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Text,
			PubDate:     item.Date.Format(time.RFC1123),
		})
	}
	rssFeed := rss.GenerateRSS(rssItems)

	w.Header().Set("Content-Type", "application/rss+xml")
	xml.NewEncoder(w).Encode(rssFeed)
}

func handleAddFeed(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	var feedInput models.Feed
	err := json.NewDecoder(r.Body).Decode(&feedInput)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := db.GetUserFromDB(userName)
	feed := db.AddFeed(feedInput, user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(
		struct {
			Created bool        `json:"created"`
			Item    models.Feed `json:"item"`
		}{
			true, feed,
		},
	)
}

func handleAddFeedLazy(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	var input struct {
		Url string `json:"url"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	feedFactory := services.FeedFactory{}
	feedInput, err := feedFactory.CreateFromUrl(input.Url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := db.GetUserFromDB(userName)
	feed := db.AddFeed(*feedInput, user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(
		struct {
			Created bool        `json:"created"`
			Item    models.Feed `json:"item"`
		}{
			true, feed,
		},
	)
}

func handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	userName, _, ok := r.BasicAuth()

	if !ok {
		w.Write([]byte("No authentication provided"))
		return
	}

	var id int
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := db.GetUserFromDB(userName)
	feed := db.GetFeedByID(id)
	if feed.UserID != user.ID {
		return
	}
	db.DeleteFeedByID(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(
		struct {
			Deleted bool        `json:"deleted"`
			Item    models.Feed `json:"item"`
		}{
			true, feed,
		},
	)
}

func handleGetItemByID(w http.ResponseWriter, r *http.Request) {
	var id int
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	item := db.GetItemByID(id)
	feed := db.GetFeedByID(item.FeedID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS,DELETE,PUT")
	json.NewEncoder(w).Encode(
		struct {
			Item models.Item `json:"item"`
			Feed models.Feed `json:"feed"`
		}{
			item, feed,
		},
	)
}
