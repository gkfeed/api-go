package main

import (
	"net/http"

	"gkfeed/api/internal/handlers"
	"gkfeed/api/pkg/auth"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/list", auth.BasicAuth(handlers.HandleListOfFeeds)).Methods("GET")
	r.HandleFunc("/api/v1/feed", auth.BasicAuth(handlers.HandleRSSFeed)).Methods("GET")
	r.HandleFunc("/api/v1/get_feed", handlers.HandleGetRSSFeed).Methods("GET")
	r.HandleFunc("/api/v1/add", auth.BasicAuth(handlers.HandleAddFeed)).Methods("POST")
	r.HandleFunc("/api/v1/delete", auth.BasicAuth(handlers.HandleDeleteFeed)).Methods("GET")
	r.HandleFunc("/api/v1/add_lazy", auth.BasicAuth(handlers.HandleAddFeedLazy)).Methods("POST")
	r.HandleFunc("/api/v1/add_deleted_items", auth.BasicAuth(handlers.HandleAddDeletedItems)).Methods("POST")
	r.HandleFunc("/api/v1/item", handlers.HandleGetItemByID).Methods("GET")
	r.HandleFunc("/api/v1/get_items", auth.BasicAuth(handlers.HandleGetItems)).Methods("GET")

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost", "http://localhost:4200", "HTTP://localhost:8086"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(r)

	http.ListenAndServe(":8086", corsHandler)
}
