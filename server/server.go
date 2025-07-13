package server

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/edamsoft-sre/alpaca/database"
	"github.com/edamsoft-sre/alpaca/models"
	"github.com/edamsoft-sre/alpaca/services"
	"github.com/edamsoft-sre/alpaca/websocket"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gorm.io/gorm"
)

type Config struct {
	Port        string
	JWTSecret   string
	DatabaseUrl string
}

type Server interface {
	Config() *Config
	Hub() *websocket.Hub
	DB() *gorm.DB
	UserService() *services.UserService
	PostService() *services.PostService
	HotelService() *services.HotelService
}

type Broker struct {
	config       *Config
	router       *mux.Router
	hub          *websocket.Hub
	db           *gorm.DB
	userService  *services.UserService
	postService  *services.PostService
	hotelService *services.HotelService
}

func (b *Broker) Config() *Config {
	return b.config
}

func (b *Broker) Hub() *websocket.Hub {
	return b.hub
}

func (b *Broker) DB() *gorm.DB {
	return b.db
}

func (b *Broker) UserService() *services.UserService {
	return b.userService
}

func (b *Broker) PostService() *services.PostService {
	return b.postService
}

func (b *Broker) HotelService() *services.HotelService {
	return b.hotelService
}

func NewServer(ctx context.Context, config *Config) (*Broker, error) {
	if config.Port == "" {
		return nil, errors.New("port is required")
	}
	if config.JWTSecret == "" {
		return nil, errors.New("jwt secret is required")
	}

	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		return nil, err
	}

	// Auto migrate models
	err = db.AutoMigrate(&models.User{}, &models.Post{}, &models.HotelAPIItem{}, &models.HotelSearchData{}, &models.HotelRatingsData{})
	if err != nil {
		return nil, err
	}

	// Initialize services
	userService := services.NewUserService(db)
	postService := services.NewPostService(db)
	hotelService := services.NewHotelService(db)

	broker := &Broker{
		config:       config,
		router:       mux.NewRouter(),
		hub:          websocket.NewHub(),
		db:           db,
		userService:  userService,
		postService:  postService,
		hotelService: hotelService,
	}
	return broker, nil
}

func (b *Broker) Start(binder func(s Server, r *mux.Router)) {
	b.router = mux.NewRouter()
	binder(b, b.router)
	handler := cors.Default().Handler(b.router)

	go b.hub.Run()
	log.Println("starting server on port", b.config.Port)
	if err := http.ListenAndServe(":"+b.config.Port, handler); err != nil {
		log.Println("error starting server:", err)
	} else {
		log.Fatalf("server stopped")
	}
}
