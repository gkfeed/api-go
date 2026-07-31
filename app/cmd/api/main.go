package main

import (
	_ "embed"
	"log"
	"net/http"

	"gkfeed/api/internal/auth"
	"gkfeed/api/internal/config"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/handlers"

	_ "gkfeed/api/cmd/api/docs"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

//go:embed test_passkey.html
var testPasskeyHTML string

// @title           GKFeed API
// @version         1.0
// @description     RSS feed aggregator with passkey authentication.

// @contact.name   GKFeed
// @contact.url    https://github.com/gkfeed/api-go

// @securityDefinitions.basic  BasicAuth

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization

func main() {
	configuration, err := config.Load()
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}
	db.Configure(configuration.DatabasePath)

	if err = db.RunMigrations(); err != nil {
		log.Fatalf("database migration: %v", err)
	}

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
	sessions := auth.NewSessionStore(configuration.AccessTokenTTL)
	authenticate := auth.AuthenticateWithSessions(configuration, sessions)

	router.PathPrefix("/api/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("doc.json"),
	))

	router.HandleFunc("/test_passkey", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(testPasskeyHTML))
	}).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/list", authenticate(handlers.HandleListOfFeeds)).Methods(http.MethodGet)
	api.HandleFunc("/feed_types", handlers.HandleListFeedTypes).Methods(http.MethodGet)
	api.HandleFunc("/feed", authenticate(handlers.HandleRSSFeed)).Methods(http.MethodGet)
	api.HandleFunc("/add", authenticate(handlers.HandleAddFeed)).Methods(http.MethodPost)
	api.HandleFunc("/delete", authenticate(handlers.HandleDeleteFeed)).Methods(http.MethodDelete)
	api.HandleFunc("/add_lazy", authenticate(handlers.HandleAddFeedLazy)).Methods(http.MethodPost)
	api.HandleFunc("/add_deleted_items", authenticate(handlers.HandleAddDeletedItems)).Methods(http.MethodPost)
	api.HandleFunc("/item", authenticate(handlers.HandleGetItemByID)).Methods(http.MethodGet)
	api.HandleFunc("/get_items", authenticate(handlers.HandleGetItems)).Methods(http.MethodGet)

	authRouter := api.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/me", authenticate(handlers.HandleMe)).Methods(http.MethodGet)

	authHandler := handlers.NewAuthHandlerWithSessions(configuration, nil, sessions)
	authRouter.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)
	authRouter.HandleFunc("/refresh", authHandler.Refresh).Methods(http.MethodPost)
	authRouter.HandleFunc("/logout", authHandler.Logout).Methods(http.MethodPost)
	authRouter.HandleFunc("/logout-all", authenticate(authHandler.LogoutAll)).Methods(http.MethodPost)

	if webAuthnService, err := auth.NewWebAuthnService(configuration); err == nil {
		authHandler = handlers.NewAuthHandlerWithSessions(configuration, webAuthnService, sessions)

		authRouter.HandleFunc("/register/begin", authenticate(authHandler.BeginRegistration)).Methods(http.MethodPost)
		authRouter.HandleFunc("/register/finish", authenticate(authHandler.FinishRegistration)).Methods(http.MethodPost)
		authRouter.HandleFunc("/login/begin", authHandler.BeginLogin).Methods(http.MethodPost)
		authRouter.HandleFunc("/login/finish", authHandler.FinishLogin).Methods(http.MethodPost)
		authRouter.HandleFunc("/credentials", authenticate(authHandler.ListCredentials)).Methods(http.MethodGet)
		authRouter.HandleFunc("/credentials/{id}", authenticate(authHandler.DeleteCredential)).Methods(http.MethodDelete)
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   configuration.AllowedOrigins,
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	return corsHandler
}
