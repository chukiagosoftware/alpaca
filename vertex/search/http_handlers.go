package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/vertex"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func LocationSelectHandler(c *gin.Context, bq *vertex.BQ) {

	var locations []vertex.LocationGroup

	locations, err := bq.GetDistinctLocations(c)

	if err != nil {
		log.Printf("error: Failed to get locations: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get locations: " + err.Error()})
	}

	c.JSON(http.StatusOK, locations)
}

func recordErrorMetric(c *gin.Context, errorType string) {
	meter := otel.Meter("vertex-search")
	counter, _ := meter.Int64Counter("search.errors")
	counter.Add(c.Request.Context(), 1, metric.WithAttributes(
		attribute.String("error_type", errorType),
	))
}

func recordVectorSearchMetrics(ctx *gin.Context, durationMs int64, resultCount int) {
	meter := otel.Meter("vertex-search")
	durationHist, _ := meter.Int64Histogram("search.vector.duration_ms")
	countGauge, _ := meter.Int64Gauge("search.vector.result_count")

	durationHist.Record(ctx, durationMs)
	countGauge.Record(ctx, int64(resultCount))
}

func buildSearchInput(form vertex.SearchForm, config *vertex.Config) vertex.SearchInput {
	var input vertex.SearchInput

	if form.Question == "" {
		input.Question = config.Query
	} else {
		input.Question = strings.TrimSpace(form.Question)
	}

	continent := strings.TrimSpace(form.Continent)
	if continent != "" {
		input.Continent = continent
	}
	if form.Rating != "" {
		if rating, err := strconv.Atoi(strings.TrimSpace(form.Rating)); err == nil && rating > 0 {
			input.Rating = rating
			input.FilterRating = true
		}
	}

	cityCountry := strings.TrimSpace(form.CityCountry)
	if cityCountry != "" {
		parts := strings.Split(cityCountry, ",")
		if len(parts) >= 2 {
			input.City = strings.TrimSpace(parts[0])
			input.Country = strings.TrimSpace(parts[1])
			input.FilterCityCountry = true
		}
	}

	log.Printf("Form inputs: continent:%s city:%s country:%s rating:%d\n", input.Continent, input.City, input.Country, input.Rating)
	return input
}

func SearchHandler(c *gin.Context, config *vertex.Config, vsSvc *vertex.VertexSearchService, bq *vertex.BQ) {

	tracer := otel.Tracer("vertex-search")
	ctx, span := tracer.Start(c.Request.Context(), "search-request")
	defer span.End()

	c.Writer.Header().Set("Content-Type", "application/json")

	var form vertex.SearchForm
	if err := c.ShouldBind(&form); err != nil {
		recordErrorMetric(c, "bind_error")
		log.Printf("Failed to bind form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	input := buildSearchInput(form, config)

	var embedTime, searchTime, safetyTime, metadataTime, completionTime time.Duration

	completionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	safetyChan := make(chan bool, 1)
	embedChan := make(chan []float32, 1)
	vectorChan := make(chan []vertex.VectorResult, 1)
	metadataChan := make(chan []map[string]any, 1)
	completionChan := make(chan string, 1)

	go func() {
		start := time.Now()
		_, safetySpan := tracer.Start(ctx, "safety-check")
		defer safetySpan.End()

		isSafe, err := vsSvc.CheckQuerySafety(ctx, *config, input.Question)
		safetyTime = time.Since(start)

		if err != nil || !isSafe {
			if err != nil {
				log.Printf("Safety check error: %v", err)
			}
			cancel()
			safetyChan <- false
			return
		}
		safetyChan <- true
	}()

	go func() {
		start := time.Now()
		_, embedSpan := tracer.Start(ctx, "embedding")
		defer embedSpan.End()

		embedding, err := vsSvc.GenerateEmbedding(ctx, input.Question)
		embedTime = time.Since(start)

		if err != nil {
			recordErrorMetric(c, "embedding_error")
			log.Printf("Embedding error: %v", err)
			embedChan <- nil
			return
		}
		embedChan <- embedding
	}()

	go func() {
		embedding := <-embedChan
		if embedding == nil {
			vectorChan <- nil
			return
		}

		start := time.Now()
		_, searchSpan := tracer.Start(ctx, "vector-search")
		results, err := vsSvc.VertexSearchEndpoint(ctx, *config, embedding, input)
		//log.Printf("Vector search results: %v", results)
		searchSpan.End()
		searchTime = time.Since(start)

		if err != nil {
			recordErrorMetric(c, "vector_search_error")
			log.Printf("Vector search error: %v", err)
			vectorChan <- nil
			return
		}
		vectorChan <- results
	}()

	go func() {
		vectorResults := <-vectorChan
		if vectorResults == nil || len(vectorResults) == 0 {
			metadataChan <- nil
			return
		}

		start := time.Now()
		_, metaSpan := tracer.Start(ctx, "metadata-lookup")

		results, err := bq.GetMetadataByIDs(ctx, vectorResults, config)
		// log.Printf("Metadata lookup results: %v", results)
		metaSpan.End()
		metadataTime = time.Since(start)

		if err != nil {
			log.Printf("Metadata lookup error: %v", err)
			metadataChan <- nil
			return
		}
		metadataChan <- results
	}()

	go func() {
		results := <-metadataChan
		if results == nil {
			completionChan <- "No relevant hotel reviews found."
			return
		}

		start := time.Now()
		_, compSpan := tracer.Start(completionCtx, "llm-completion")
		defer compSpan.End()

		completion, err := vsSvc.PromptCompletion(completionCtx, *config, input.Question, results)
		completionTime = time.Since(start)

		if err != nil {
			log.Printf("Completion error: %v", err)
			completionChan <- "Sorry, I could not generate a response."
			return
		}
		completionChan <- completion
	}()

	isSafe := <-safetyChan
	completion := <-completionChan

	if !isSafe {
		completion = "Your query was flagged as not relevant to hotel reviews. Please try a different question."
	}

	finalResults := []map[string]any{}
	// We don't block on metadataChan again if safety already failed, but for timing we can read it safely
	select {
	case finalResults = <-metadataChan:
	default:
	}

	parsedReviews, parseErr := parseCompletionJSON(completion)
	if parseErr != nil {
		log.Printf("Failed to parse LLM JSON: %v", parseErr)
		parsedReviews = []map[string]any{}
	}

	c.JSON(http.StatusOK, gin.H{
		"completion":   parsedReviews,
		"vector_count": len(finalResults),
		"safe_query":   isSafe,
		"timings": gin.H{
			"embedding_ms":      embedTime.Milliseconds(),
			"vector_search_ms":  searchTime.Milliseconds(),
			"safety_ms":         safetyTime.Milliseconds(),
			"metadata_ms":       metadataTime.Milliseconds(),
			"llm_completion_ms": completionTime.Milliseconds(),
		},
	})
}

func Pong(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func parseCompletionJSON(completionStr string) ([]map[string]any, error) {
	trimmedSpace := strings.TrimSpace(completionStr)
	// Remove markdown code blocks if they somehow appear
	dirtyRegexp := regexp.MustCompile("(?s)^```(?:json)?\\s*|\\s*```$")
	clean := dirtyRegexp.ReplaceAllString(trimmedSpace, "")

	var err error
	var reviews []map[string]any
	if err = json.Unmarshal([]byte(clean), &reviews); err == nil {
		log.Println("Successfully parsed LLM returned JSON")
		return reviews, nil
	}

	log.Printf("Failed to unmarshal LLM returned JSON. Trying backtick removal %v\n", err)
	noBackTick := strings.ReplaceAll(trimmedSpace, "```", "")
	noFrontTick := strings.ReplaceAll(noBackTick, "json```", "")
	if err = json.Unmarshal([]byte(noFrontTick), &reviews); err == nil {
		log.Println("Successfully parsed LLM returned JSON after removing backticks")
		return reviews, nil
	}
	log.Printf("Failed to parse LLM returned JSON after backtick removal: %v\n", err)
	return nil, err
}
