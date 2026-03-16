package main

import (
	"log"
	"net/http"
	"time"

	"github.com/chukiagosoftware/alpaca/vertex"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

func SearchHandler(c *gin.Context, config *vertex.Config, vsSvc *vertex.VertexSearchService) {
	tracer := otel.Tracer("vertex-search")

	// Extract form data
	question := c.PostForm("question")
	if question == "" {
		question = config.Query // Default from config
	}
	city := c.PostForm("city")
	log.Printf("City: %s \n", city)

	// Overall span for the request
	ctx, span := tracer.Start(c.Request.Context(), "search-request")
	defer span.End()

	// Channels for pipeline (buffered to avoid blocking)
	embedChan := make(chan []float32, 1)
	searchChan := make(chan []map[string]any, 1)
	completionChan := make(chan string, 1)

	// Timing vars for display
	var embedTime, searchTime, completionTime time.Duration

	go func() {
		_, embedSpan := tracer.Start(ctx, "embedding")
		start := time.Now()
		// Use timeout context for the call
		embedding, err := vsSvc.GenerateEmbedding(ctx, question)
		embedTime = time.Since(start)
		embedSpan.End()
		if err != nil {
			log.Printf("Embedding error: %v", err)
			embedChan <- nil
			return
		}

		embedChan <- embedding
	}()

	go func() {
		embedding := <-embedChan
		if embedding == nil {
			searchChan <- nil
			return
		}
		_, searchSpan := tracer.Start(ctx, "vector-search")
		start := time.Now()
		results, err := vsSvc.VertexSearchEndpoint(ctx, *config, embedding, config.Limit)
		searchTime = time.Since(start)
		searchSpan.End()
		if err != nil {
			log.Printf("Vector search error: %v", err)
			searchChan <- nil
			return
		}
		// Todo Filter results in vector search first
		//if city != "" && city != "All Cities" {
		//	log.Println("Filtering results by city")
		//	filtered := []map[string]any{}
		//	for _, result := range results {
		//		if result["city"].(string) == city {
		//			filtered = append(filtered, result)
		//		}
		//	}
		//	results = filtered
		//}
		searchChan <- results

	}()

	go func() {
		results := <-searchChan
		if results == nil {
			completionChan <- "Error in processing"
			return
		}
		_, completionSpan := tracer.Start(ctx, "llm-completion")
		start := time.Now()
		completion, err := vsSvc.PromptCompletion(ctx, *config, results)
		log.Println(completion)
		completionTime = time.Since(start)
		completionSpan.End()
		if err != nil {
			log.Printf("Completion error: %v", err)
			completionChan <- "Error generating completion"
			return
		}
		completionChan <- completion
	}()

	completion := <-completionChan
	// Return JSON with results and timings
	c.JSON(http.StatusOK, gin.H{
		"completion": completion,
		"timings": gin.H{
			"embedding_ms":      embedTime.Milliseconds(),
			"vector_search_ms":  searchTime.Milliseconds(),
			"llm_completion_ms": completionTime.Milliseconds(),
		},
	})

}

func Pong(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
