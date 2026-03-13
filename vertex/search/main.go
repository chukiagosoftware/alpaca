package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/chukiagosoftware/alpaca/vertex"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectID  string    `yaml:"project_id"`
	Location   string    `yaml:"location"`
	DatasetID  string    `yaml:"dataset_id"`
	IndexID    string    `yaml:"index_id"`
	EndpointID string    `yaml:"endpoint_id"`
	Mode       string    `yaml:"mode"`
	Force      bool      `yaml:"force"`
	Query      []float64 `yaml:"query"`
	Limit      int       `yaml:"limit"`
}

func loadConfig() (*Config, error) {
	config := &Config{}
	if file, err := os.Open("config.yaml"); err == nil {
		defer file.Close()
		if err := yaml.NewDecoder(file).Decode(config); err != nil {
			return nil, err
		}
	}
	if v := os.Getenv("PROJECT_ID"); v != "" {
		config.ProjectID = v
	}
	if v := os.Getenv("LOCATION"); v != "" {
		config.Location = v
	}
	if v := os.Getenv("DATASET_ID"); v != "" {
		config.DatasetID = v
	}
	if v := os.Getenv("INDEX_ID"); v != "" {
		config.IndexID = v
	}
	if v := os.Getenv("ENDPOINT_ID"); v != "" {
		config.EndpointID = v
	}
	if v := os.Getenv("MODE"); v != "" {
		config.Mode = v
	}
	if v := os.Getenv("FORCE"); v == "true" {
		config.Force = true
	}
	if v := os.Getenv("LIMIT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			config.Limit = i
		}
	}
	return config, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	log.Printf("Loaded config: ProjectID=%s, Location=%s, DatasetID=%s, IndexID=%s, EndpointID=%s, Mode=%s, Force=%t, Limit=%d",
		config.ProjectID, config.Location, config.DatasetID, config.IndexID, config.EndpointID, config.Mode, config.Force, config.Limit)

	ctx := context.Background()
	bqSvc, err := vertex.NewBigQueryService(ctx, config.ProjectID, config.DatasetID)
	if err != nil {
		log.Fatal("Failed to create BQ service:", err)
	}
	defer bqSvc.Close()

	vsSvc, err := vertex.NewVertexSearchService(ctx, config.ProjectID, config.Location, bqSvc)
	if err != nil {
		log.Fatal("Failed to create Vertex service:", err)
	}

	switch config.Mode {
	case "1":
		if err := bqSvc.GenerateEmbeddingsForHotelReviews(ctx, config.Force); err != nil {
			log.Fatal("Failed to generate embeddings:", err)
		}
		fmt.Println("Embeddings generated")
	case "2":
		results, err := bqSvc.SearchSimilarReviewsBQ(ctx, config.Query, config.Limit)
		if err != nil {
			log.Fatal("Failed to search BQ:", err)
		}
		fmt.Printf("BQ Results: %+v\n", results)
	case "3a":
		if err := vsSvc.UploadEmbeddingsToVertexSearch(ctx, config.IndexID); err != nil {
			log.Fatal("Failed to upload to Vertex:", err)
		}
		fmt.Println("Uploaded to Vertex index")
	case "3b":
		results, err := vsSvc.SearchVertex(ctx, config.IndexID, config.Query, config.Limit)
		if err != nil {
			log.Fatal("Failed to search Vertex index:", err)
		}
		fmt.Printf("Vertex Index Results: %+v\n", results)
	case "3c":
		if _, err := vsSvc.DeployEndpoint(ctx, config.IndexID, config.EndpointID); err != nil {
			log.Fatal("Failed to deploy endpoint:", err)
		}
		results, err := vsSvc.SearchVertexEndpoint(ctx, config.EndpointID, config.Query, config.Limit)
		if err != nil {
			log.Fatal("Failed to search endpoint:", err)
		}
		fmt.Printf("Vertex Endpoint Results: %+v\n", results)
	default:
		log.Fatal("Invalid mode")
	}
}
