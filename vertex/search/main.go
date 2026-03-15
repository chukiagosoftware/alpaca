package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chukiagosoftware/alpaca/vertex"
)

func main() {
	config, err := vertex.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Loaded config: ProjectID=%s\n, Location=%s\n, DatasetID=%s\n, IndexID=%s\n, EndpointID=%s\n, Domain=%s\n",
		config.ProjectID,
		config.Location,
		config.DatasetID,
		config.IndexID,
		config.EndpointID,
		config.EndpointPublicDomainName)

	baseCtx := context.Background()
	ctx, cancel := context.WithTimeout(baseCtx, 60*time.Second)
	defer cancel() // It's important to call cancel to release resources

	vsSvc, err := vertex.NewVertexSearchService(ctx, config)
	if err != nil {
		log.Fatal("Failed to create Vertex service:", err)
	}
	defer vsSvc.Close()

	results, err := vsSvc.QuerySimilarReviews(ctx, *config)
	if err != nil {
		log.Fatal("Failed to search endpoint:", err)
	}
	fmt.Println("Similar reviews:")
	for _, result := range results {
		fmt.Println(result)
		fmt.Println()
	}

	recommendation, err := vsSvc.PromptCompletion(ctx, *config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(recommendation)

}
