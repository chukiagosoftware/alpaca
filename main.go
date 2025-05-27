package main

import (
	"fmt"
	"context"
	"log"
	"net/http"
	"os"

	"github.com/edamsoft-sre/alpaca/handlers"
	"github.com/edamsoft-sre/alpaca/server"
	"github.com/edamsoft-sre/alpaca/middleware"
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
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	PORT := os.Getenv("PORT")
	DATABASE_URL := os.Getenv("DATABASE_URL")
	JWT_SECRET := os.Getenv("JWT_SECRET")
	AMD := os.Getenv("AMD")
	AMS := os.Getenv("AMS")

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
	r.HandleFunc("/auth/{provider}/callback", handlers.GothCallbackHandler)

	r.HandleFunc("/api/v1/signup", handlers.SignUpHandler(s)).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/login", handlers.LoginHandler(s)).Methods(http.MethodPost)
	api.HandleFunc("/me", handlers.MeHandler(s)).Methods(http.MethodGet)

	api.HandleFunc("/posts", handlers.InsertPostHandler(s)).Methods(http.MethodPost)
	api.HandleFunc("/posts/{postId}", handlers.DeletePostByIdHandler(s)).Methods(http.MethodDelete)
	api.HandleFunc("/posts/{postId}", handlers.UpdatePostByIdHandler(s)).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/posts/{postId}", handlers.GetPostByIDHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/posts", handlers.ListPostHandler(s)).Methods(http.MethodGet)


	r.HandleFunc("/ws", s.Hub().HandleWebSocket)

	r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    r.HandleFunc("/topology", handlers.TopologyHandler).Methods("GET")


}
