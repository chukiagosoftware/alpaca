package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/edamsoft-sre/alpaca/handlers"
	"github.com/edamsoft-sre/alpaca/middleware"
	"github.com/edamsoft-sre/alpaca/server"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type PageData struct {
	PythonCode string
	GoCode     string
	Pods       []string
	KafkaTopic string
}

func main() {
	// Smart .env loading - try multiple locations
	envLoaded := false
	envPaths := []string{
		".env",       // Current directory
		"../.env",    // Parent directory
		"../../.env", // Grandparent directory
	}

	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env file from: %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Printf("Warning: No .env file found in any of the expected locations")
		log.Printf("Make sure environment variables are set manually")
	}

	PORT := os.Getenv("PORT")
	DATABASE_URL := os.Getenv("DATABASE_URL")
	JWT_SECRET := os.Getenv("JWT_SECRET")

	// Validate required environment variables
	if PORT == "" {
		PORT = "8080" // Default port
		log.Printf("PORT not set, using default: %s", PORT)
	}
	if JWT_SECRET == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	s, err := server.NewServer(context.Background(), &server.Config{
		Port:        PORT,
		JWTSecret:   JWT_SECRET,
		DatabaseUrl: DATABASE_URL,
	})

	if err != nil {
		log.Fatal(err)
		fmt.Println("This is it")
	}

	handlers.InitGoth()

	s.Start(BindRoutes)
}

func BindRoutes(s server.Server, r *mux.Router) {

	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.CheckAuthMiddleware(s))

	r.HandleFunc("/", handlers.HomeHandler(s)).Methods(http.MethodGet)

	r.HandleFunc("/auth/{provider}/login", handlers.GothLoginHandler)
	r.HandleFunc("/auth/{provider}/callback", handlers.GothCallbackHandler(s))

	r.HandleFunc("/api/v1/signup", handlers.SignUpHandler(s)).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/login", handlers.LoginHandler(s)).Methods(http.MethodPost)
	api.HandleFunc("/me", handlers.MeHandler(s)).Methods(http.MethodGet)

	api.HandleFunc("/posts", handlers.InsertPostHandler(s)).Methods(http.MethodPost)
	api.HandleFunc("/posts/{postId}", handlers.DeletePostByIdHandler(s)).Methods(http.MethodDelete)
	api.HandleFunc("/posts/{postId}", handlers.UpdatePostByIdHandler(s)).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/posts/{postId}", handlers.GetPostByIDHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/posts", handlers.ListPostHandler(s)).Methods(http.MethodGet)

	// Hotel routes (public - no auth required)
	r.HandleFunc("/api/v1/hotels", handlers.ListHotelsHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/hotels/{hotelId}", handlers.GetHotelByIDHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/hotels/city/{cityName}", handlers.GetHotelsByCityHandler(s)).Methods(http.MethodGet)

	// New hotel routes with relationship data
	r.HandleFunc("/api/v1/hotels/complete", handlers.ListHotelsWithCompleteDataHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/hotels/with-search", handlers.ListHotelsWithSearchDataHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/hotels/with-ratings", handlers.ListHotelsWithRatingsDataHandler(s)).Methods(http.MethodGet)

	r.HandleFunc("/ws", s.Hub().HandleWebSocket)

	r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.HandleFunc("/topology", handlers.TopologyHandler).Methods("GET")

}
