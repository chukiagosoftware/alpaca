package aisearch

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	datasetID := os.Getenv("BQ_DATASET_ID")
	if projectID == "" || datasetID == "" {
		log.Fatal("GCP_PROJECT_ID and BQ_DATASET_ID must be set")
	}

	service, err := aisearch.NewAISearchService(projectID, datasetID)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	handler := aisearch.NewHandler(service)

	r := gin.Default()
	handler.SetupRoutes(r)

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
