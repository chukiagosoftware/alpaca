package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/vertex"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

func SearchHandler(c *gin.Context, config *vertex.Config, vsSvc *vertex.VertexSearchService) {
	tracer := otel.Tracer("vertex-search")

	// Extract form data
	question := strings.TrimSpace(c.PostForm("question"))
	if question == "" {
		question = config.Query // Default from config
	}
	city := strings.TrimSpace(c.PostForm("city"))
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
		results, err := vsSvc.VertexSearchEndpoint(ctx, *config, embedding, city)
		searchTime = time.Since(start)
		searchSpan.End()
		if err != nil {
			log.Printf("Vector search error: %v", err)
			searchChan <- nil
			return
		}
		// log.Println(results)
		searchChan <- results

	}()

	go func() {
		similarityResults := <-searchChan
		if similarityResults == nil {
			completionChan <- "No Vector Similarity Results"
			return
		}
		_, completionSpan := tracer.Start(ctx, "llm-completion")
		start := time.Now()
		completion, err := vsSvc.PromptCompletion(ctx, *config, question, similarityResults)
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
