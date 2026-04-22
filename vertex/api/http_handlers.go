package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/vertex"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func LocationSelectHandler(c *gin.Context, bq *BQ) {

	var locations []LocationGroup

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

func recordLLMMetrics(c *gin.Context, model string, usage vertex.TokenUsage, success bool, isSafe bool) {
	meter := otel.Meter("vertex-search")

	// Track which model was used
	modelCounter, _ := meter.Int64Counter("llm.model.usage")
	modelCounter.Add(c.Request.Context(), 1, metric.WithAttributes(
		attribute.String("model", model),
		attribute.Bool("success", success),
	))

	if usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
		tokenHist, _ := meter.Int64Histogram("llm.tokens")
		tokenHist.Record(c.Request.Context(), int64(usage.PromptTokens),
			metric.WithAttributes(attribute.String("type", "prompt"), attribute.String("model", model)))
		tokenHist.Record(c.Request.Context(), int64(usage.CompletionTokens),
			metric.WithAttributes(attribute.String("type", "completion"), attribute.String("model", model)))
	}

	completionCounter, _ := meter.Int64Counter("llm.completion.count")
	completionCounter.Add(c.Request.Context(), 1, metric.WithAttributes(
		attribute.Bool("success", success),
		attribute.Bool("safe_query", isSafe),
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

	if form.LLMChoice != "" && strings.ToLower(strings.TrimSpace(form.LLMChoice)) != "auto" {
		input.PreferredModel = form.LLMChoice
	} else {
		input.PreferredModel = config.PreferredModel
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

func SearchHandler(c *gin.Context, config *vertex.Config, vsSvc *vertex.VertexSearchService, bq *BQ) {

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
	vectorCountChan := make(chan int, 1)
	metadataChan := make(chan []map[string]any, 1)
	completionChan := make(chan vertex.CompletionResult, 1)

	go func() {
		start := time.Now()
		_, safetySpan := tracer.Start(ctx, "safety-check")
		defer safetySpan.End()

		isSafe, err := vsSvc.CheckQuerySafety(ctx, input)
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
			vectorCountChan <- 0
			return
		}

		start := time.Now()
		_, searchSpan := tracer.Start(ctx, "vector-search")
		results, err := vsSvc.VertexSearchEndpoint(ctx, *config, embedding, input)
		//log.Printf("Vector search results: %v", results)
		searchSpan.End()
		searchTime = time.Since(start)

		count := 0
		if results != nil {
			count = len(results)
		}

		if err != nil {
			recordErrorMetric(c, "vector_search_error")
			log.Printf("Vector search error: %v", err)
			vectorChan <- nil
			vectorCountChan <- 0
			return
		}
		vectorChan <- results
		vectorCountChan <- count
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
			completionChan <- vertex.CompletionResult{Content: "[]"}
			return
		}

		if completionCtx.Err() != nil {
			completionChan <- vertex.CompletionResult{Content: "[]"}
			return
		}

		start := time.Now()
		_, compSpan := tracer.Start(completionCtx, "llm-completion")
		defer compSpan.End()

		completion, err := vsSvc.PromptCompletion(completionCtx, input, results)
		completionTime = time.Since(start)

		if err != nil {
			log.Printf("Completion error: %v", err)
			completionChan <- vertex.CompletionResult{Content: "[]"}
			return
		}
		completionChan <- completion
	}()

	isSafe := <-safetyChan
	compResult := <-completionChan

	var userMessage string
	if !isSafe {
		userMessage = "Your query was flagged as not relevant to hotel reviews. Please try a different question."
		compResult = vertex.CompletionResult{Content: "Your query was flagged as not relevant to hotel reviews. Please try a different question."}
	}

	vectorCount := 0
	select {
	case vectorCount = <-vectorCountChan:
	default:
	}

	parsedReviews, parseErr := parseCompletionJSON(compResult.Content)
	if parseErr != nil {
		log.Printf("Failed to parse LLM JSON: %v", parseErr)
		parsedReviews = []map[string]any{}
		recordErrorMetric(c, "json_parse_error")
	}

	googleKey := config.GooglePlacesAPIKey
	if googleKey == "" {
		log.Println("Warning: GOOGLE_MAPS_API_KEY not set - maps/photos skipped")
	} else {
		for i := range parsedReviews {
			enrichReviewWithGoogleMedia(&parsedReviews[i], googleKey)
		}
	}

	recordLLMMetrics(c, compResult.Model, compResult.Usage, len(parsedReviews) > 0 || userMessage != "", isSafe)
	recordVectorSearchMetrics(c, searchTime.Milliseconds(), vectorCount)

	c.JSON(http.StatusOK, gin.H{
		"completion":   parsedReviews,
		"message":      userMessage,
		"model":        compResult.Model,
		"usage":        compResult.Usage,
		"vector_count": vectorCount,
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
