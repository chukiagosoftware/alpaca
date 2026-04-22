package main

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	config := cors.DefaultConfig()

	if gin.Mode() == gin.DebugMode {
		// Development (Vite dev server)
		config.AllowOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
		config.AllowCredentials = true
	} else {
		// Production - much more restrictive
		// If serving built frontend from same origin, you can even disable CORS here
		config.AllowOrigins = []string{} // Add your real domain(s) if needed
		config.AllowCredentials = false
	}

	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.MaxAge = 12 * time.Hour

	return cors.New(config)
}
