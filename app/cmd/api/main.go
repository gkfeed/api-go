package main

import (
	_ "embed"
	"log"
	"net/http"

	authsvc "gkfeed/api/internal/auth"
	"gkfeed/api/internal/config"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/handlers"
	"gkfeed/api/pkg/auth"

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
// @description                JWT token: "Bearer <token>"

func main() {
	configuration := config.Load()
	db.Configure(configuration.DatabasePath)

	if err := db.RunMigrations(); err != nil {
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
	authenticate := auth.Authenticate(configuration)
	jwtAuth := auth.JWTAuth(configuration)

	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	router.HandleFunc("/test_passkey", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(testPasskeyHTML))
	}).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/list", authenticate(handlers.HandleListOfFeeds)).Methods(http.MethodGet)
	api.HandleFunc("/feed", authenticate(handlers.HandleRSSFeed)).Methods(http.MethodGet)
	api.HandleFunc("/add", authenticate(handlers.HandleAddFeed)).Methods(http.MethodPost)
	api.HandleFunc("/delete", authenticate(handlers.HandleDeleteFeed)).Methods(http.MethodDelete)
	api.HandleFunc("/add_lazy", authenticate(handlers.HandleAddFeedLazy)).Methods(http.MethodPost)
	api.HandleFunc("/add_deleted_items", authenticate(handlers.HandleAddDeletedItems)).Methods(http.MethodPost)
	api.HandleFunc("/item", handlers.HandleGetItemByID).Methods(http.MethodGet)
	api.HandleFunc("/get_items", authenticate(handlers.HandleGetItems)).Methods(http.MethodGet)

	if waSvc, err := authsvc.NewService(configuration); err == nil {
		ah := handlers.NewAuthHandler(configuration, waSvc)

		authRouter := router.PathPrefix("/auth").Subrouter()
		authRouter.HandleFunc("/register/begin", authenticate(ah.BeginRegistration)).Methods(http.MethodPost)
		authRouter.HandleFunc("/register/finish", authenticate(ah.FinishRegistration)).Methods(http.MethodPost)
		authRouter.HandleFunc("/login/begin", ah.BeginLogin).Methods(http.MethodPost)
		authRouter.HandleFunc("/login/finish", ah.FinishLogin).Methods(http.MethodPost)
		authRouter.HandleFunc("/refresh", ah.Refresh).Methods(http.MethodPost)
		authRouter.HandleFunc("/logout", authenticate(ah.Logout)).Methods(http.MethodPost)
		authRouter.HandleFunc("/credentials", authenticate(ah.ListCredentials)).Methods(http.MethodGet)
		authRouter.HandleFunc("/credentials/{id}", authenticate(ah.DeleteCredential)).Methods(http.MethodDelete)
		authRouter.HandleFunc("/me", jwtAuth(ah.Me)).Methods(http.MethodGet)
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   configuration.AllowedOrigins,
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	return corsHandler
}
