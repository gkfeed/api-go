package main

import (
	"log"
	"net/http"

	"gkfeed/api/internal/config"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/handlers"
	"gkfeed/api/pkg/auth"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	configuration := config.Load()
	db.Configure(configuration.DatabasePath)

	server := &http.Server{
		Addr:              configuration.Address,
		Handler:           newHandler(configuration),
		ReadHeaderTimeout: configuration.ReadHeaderTimeout,
	}
	log.Printf("listening on %s", configuration.Address)
	log.Fatal(server.ListenAndServe())
}

func newHandler(configuration config.Config) http.Handler {
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/list", auth.BasicAuth(handlers.HandleListOfFeeds)).Methods(http.MethodGet)
	api.HandleFunc("/feed", auth.BasicAuth(handlers.HandleRSSFeed)).Methods(http.MethodGet)
	api.HandleFunc("/add", auth.BasicAuth(handlers.HandleAddFeed)).Methods(http.MethodPost)
	api.HandleFunc("/delete", auth.BasicAuth(handlers.HandleDeleteFeed)).Methods(http.MethodGet, http.MethodDelete)
	api.HandleFunc("/add_lazy", auth.BasicAuth(handlers.HandleAddFeedLazy)).Methods(http.MethodPost)
	api.HandleFunc("/add_deleted_items", auth.BasicAuth(handlers.HandleAddDeletedItems)).Methods(http.MethodPost)
	api.HandleFunc("/item", handlers.HandleGetItemByID).Methods(http.MethodGet)
	api.HandleFunc("/get_items", auth.BasicAuth(handlers.HandleGetItems)).Methods(http.MethodGet)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   configuration.AllowedOrigins,
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	return corsHandler
}
