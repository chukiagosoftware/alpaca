package aisearch

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/vertexai/genai"
	"google.golang.org/api/iterator"
)

// AISearchService handles the AI search operations
type AISearchService struct {
	bqClient    *bigquery.Client
	genaiClient *genai.Client
	projectID   string
	datasetID   string
}

// NewAISearchService creates a new service
func NewAISearchService(projectID, datasetID string) (*AISearchService, error) {
	ctx := context.Background()
	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	genaiClient, err := genai.NewClient(ctx, projectID, "us-central1")
	if err != nil {
		return nil, err
	}

	return &AISearchService{
		bqClient:    bqClient,
		genaiClient: genaiClient,
		projectID:   projectID,
		datasetID:   datasetID,
	}, nil
}

// UploadCities uploads cities data to BigQuery
func (s *AISearchService) UploadCities(ctx context.Context, cities []City) error {
	table := s.bqClient.Dataset(s.datasetID).Table("cities")
	uploader := table.Uploader()
	return uploader.Put(ctx, cities)
}

// Similar for hotels and reviews
func (s *AISearchService) UploadHotels(ctx context.Context, hotels []Hotel) error {
	table := s.bqClient.Dataset(s.datasetID).Table("hotels")
	uploader := table.Uploader()
	return uploader.Put(ctx, hotels)
}

func (s *AISearchService) UploadReviews(ctx context.Context, reviews []Review) error {
	table := s.bqClient.Dataset(s.datasetID).Table("reviews")
	uploader := table.Uploader()
	return uploader.Put(ctx, reviews)
}

// TransformData transforms the data into Star schema
func (s *AISearchService) TransformData(ctx context.Context) error {
	// Fetch data from tables
	cities, err := s.fetchCities(ctx)
	if err != nil {
		return err
	}
	hotels, err := s.fetchHotels(ctx)
	if err != nil {
		return err
	}
	reviews, err := s.fetchReviews(ctx)
	if err != nil {
		return err
	}

	// Build maps
	cityMap := make(map[string]City)
	for _, c := range cities {
		cityMap[c.IATACode] = c
	}

	hotelMap := make(map[int64]Hotel)
	for _, h := range hotels {
		hotelMap[h.ID] = h
	}

	reviewMap := make(map[int64][]Review)
	for _, r := range reviews {
		reviewMap[r.HotelID] = append(reviewMap[r.HotelID], r)
	}

	// Enrich
	var starHotels []StarHotel
	for _, h := range hotels {
		city, ok := cityMap[h.IATACode]
		if !ok {
			continue
		}
		reviews := reviewMap[h.ID]
		star := StarHotel{
			City:               city.Name,
			NearestAirportCode: city.IATACode,
			Latitude:           h.Latitude,
			Longitude:          h.Longitude,
			HotelName:          h.Name,
			Address:            h.Address,
			GoogleRating:       h.GoogleRating,
			OverallRating:      calculateOverallRating(reviews),
			QualityRating:      calculateQualityRating(reviews),
			QuietRating:        calculateQuietRating(reviews),
			AdminOverride:      "", // not implemented
		}
		starHotels = append(starHotels, star)
	}

	// Insert to star_hotels table
	table := s.bqClient.Dataset(s.datasetID).Table("star_hotels")
	uploader := table.Uploader()
	return uploader.Put(ctx, starHotels)
}

// Helper functions for calculations
func calculateOverallRating(reviews []Review) float64 {
	if len(reviews) == 0 {
		return 0
	}
	sum := 0
	for _, r := range reviews {
		sum += r.Rating
	}
	return float64(sum) / float64(len(reviews))
}

func calculateQualityRating(reviews []Review) float64 {
	// Assume quality is average of rating
	return calculateOverallRating(reviews)
}

func calculateQuietRating(reviews []Review) float64 {
	if len(reviews) == 0 {
		return 0
	}
	sum := 0
	for _, r := range reviews {
		sum += r.Quiet
	}
	return float64(sum) / float64(len(reviews))
}

// Fetch functions
func (s *AISearchService) fetchCities(ctx context.Context) ([]City, error) {
	q := s.bqClient.Query("SELECT * FROM `" + s.datasetID + ".cities`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	var cities []City
	for {
		var c City
		err := it.Next(&c)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		cities = append(cities, c)
	}
	return cities, nil
}

// Similar for hotels and reviews
func (s *AISearchService) fetchHotels(ctx context.Context) ([]Hotel, error) {
	q := s.bqClient.Query("SELECT * FROM `" + s.datasetID + ".hotels`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	var hotels []Hotel
	for {
		var h Hotel
		err := it.Next(&h)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		hotels = append(hotels, h)
	}
	return hotels, nil
}

func (s *AISearchService) fetchReviews(ctx context.Context) ([]Review, error) {
	q := s.bqClient.Query("SELECT * FROM `" + s.datasetID + ".reviews`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	var reviews []Review
	for {
		var r Review
		err := it.Next(&r)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, nil
}

// VectorizeData vectorizes the star hotels and stores embeddings
func (s *AISearchService) VectorizeData(ctx context.Context) error {
	// Fetch star hotels
	starHotels, err := s.fetchStarHotels(ctx)
	if err != nil {
		return err
	}

	// For each, generate text and embed
	for i, sh := range starHotels {
		text := fmt.Sprintf("Hotel: %s, City: %s, Address: %s, Rating: %.1f", sh.HotelName, sh.City, sh.Address, sh.GoogleRating)
		emb, err := s.generateEmbedding(ctx, text)
		if err != nil {
			log.Printf("Error embedding %s: %v", sh.HotelName, err)
			continue
		}
		starHotels[i].Embedding = emb
	}

	// Update table with embeddings
	table := s.bqClient.Dataset(s.datasetID).Table("star_hotels")
	uploader := table.Uploader()
	return uploader.Put(ctx, starHotels)
}

// fetchStarHotels
func (s *AISearchService) fetchStarHotels(ctx context.Context) ([]StarHotel, error) {
	q := s.bqClient.Query("SELECT * FROM `" + s.datasetID + ".star_hotels`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	var starHotels []StarHotel
	for {
		var sh StarHotel
		err := it.Next(&sh)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		starHotels = append(starHotels, sh)
	}
	return starHotels, nil
}

// generateEmbedding uses Vertex AI to generate embedding
func (s *AISearchService) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	em := s.genaiClient.EmbeddingModel("text-embedding-004")
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}
	if len(res.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding values")
	}
	return res.Embedding.Values, nil
}

// VectorSearch performs vector search
func (s *AISearchService) VectorSearch(ctx context.Context, query string) ([]StarHotel, error) {
	// Generate query embedding
	queryEmb, err := s.generateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	// In BigQuery, use VECTOR_SEARCH or approximate with cosine similarity
	// For simplicity, fetch all and compute similarity
	starHotels, err := s.fetchStarHotels(ctx)
	if err != nil {
		return nil, err
	}

	// Compute cosine similarity
	type scoredHotel struct {
		hotel StarHotel
		score float64
	}
	var scored []scoredHotel
	for _, sh := range starHotels {
		if len(sh.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(queryEmb, sh.Embedding)
		scored = append(scored, scoredHotel{sh, score})
	}

	// Sort by score desc
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top 10
	var results []StarHotel
	for i, sc := range scored {
		if i >= 10 {
			break
		}
		results = append(results, sc.hotel)
	}
	return results, nil
}

// cosineSimilarity
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, norma, normb float64
	for i := range a {
		dot += float64(a[i] * b[i])
		norma += float64(a[i] * a[i])
		normb += float64(b[i] * b[i])
	}
	if norma == 0 || normb == 0 {
		return 0
	}
	return dot / (math.Sqrt(norma) * math.Sqrt(normb))
}

// GenerateLLMResponse uses Gemini to generate response
func (s *AISearchService) GenerateLLMResponse(ctx context.Context, req LLMRequest) (string, error) {
	model := s.genaiClient.GenerativeModel("gemini-1.5-flash")

	// Build prompt
	prompt := req.Prompt + "\n\nBased on the following hotel data:\n"
	for _, sh := range req.SearchResults {
		prompt += fmt.Sprintf("- %s in %s, rating %.1f\n", sh.HotelName, sh.City, sh.GoogleRating)
	}
	prompt += "\nAnswer the query: " + req.Query

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "No response generated", nil
	}
	return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
}
