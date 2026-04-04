package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/vertex"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

func LocationSelectHandler(c *gin.Context, config *vertex.Config, bq *vertex.BQ) {

	var locations []vertex.LocationGroup

	locations, err := bq.GetDistinctLocations(c)

	if err != nil {
		log.Printf("error: Failed to get locations: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get locations: " + err.Error()})
	}

	c.JSON(http.StatusOK, locations)
}

func SearchHandler(c *gin.Context, config *vertex.Config, vsSvc *vertex.VertexSearchService) {

	tracer := otel.Tracer("vertex-search")
	question := strings.TrimSpace(c.PostForm("question"))
	if question == "" {
		question = config.Query
	}
	continent := strings.TrimSpace(c.PostForm("continent"))
	cityCountry := strings.TrimSpace(c.PostForm("citycountry"))
	rating, err := strconv.Atoi(strings.TrimSpace(c.PostForm("rating")))
	log.Printf("Continent: %s, CityCountry: %s, Rating: %d \n", continent, cityCountry, rating)
	if err != nil {
		rating = 0
	}

	var searchParams vertex.SearchInput

	if continent != "All Regions" {
		searchParams.Continent = continent
	}

	if rating != 0 {
		searchParams.Rating = rating
		searchParams.FilterRating = true
	}

	if cityCountry != "All Cities" {
		searchParams.City = strings.Split(cityCountry, ",")[0]
		searchParams.Country = strings.Split(cityCountry, ",")[1]
		searchParams.FilterCityCountry = true
	}

	log.Printf("SearchParams: %v \n", searchParams)

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
		results, err := vsSvc.VertexSearchEndpoint(ctx, *config, embedding, searchParams)
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
