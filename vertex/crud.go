package vertex

import (
	"context"
	"fmt"
	"math"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/chukiagosoftware/alpaca/models"
	"google.golang.org/api/iterator"
)

// BQ handles BigQuery operations
type BQ struct {
	BQClient  *bigquery.Client
	ProjectID string
	DatasetID string
}

// NewBigQueryService creates a new service
func NewBigQueryService(ctx context.Context, projectID, datasetID string) (*BQ, error) {
	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &BQ{
		BQClient:  bqClient,
		ProjectID: projectID,
		DatasetID: datasetID,
	}, nil
}

// Close closes the BigQuery client
func (s *BQ) Close() error {
	return s.BQClient.Close()
}

// CreateBigQueryTable creates the table with schema inferred from the struct
func (s *BQ) CreateBigQueryTable(ctx context.Context, inferStruct interface{}, tableName string) error {
	schema, err := bigquery.InferSchema(inferStruct)
	if err != nil {
		return err
	}
	table := s.BQClient.Dataset(s.DatasetID).Table(tableName)
	err = table.Create(ctx, &bigquery.TableMetadata{
		Schema: schema,
	})
	// Ignore "already exists" errors
	if err != nil && strings.Contains(err.Error(), "Already Exists") {
		return nil
	}
	return err
}

// UploadData uploads data to BigQuery in batches
func UploadData[T any](ctx context.Context, s *BQ, tableName string, data []T) error {
	if len(data) == 0 {
		return nil
	}

	table := s.BQClient.Dataset(s.DatasetID).Table(tableName)
	inserter := table.Inserter()

	const batchSize = 500
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		batch := data[i:end]
		if err := inserter.Put(ctx, batch); err != nil {
			return fmt.Errorf("failed to upload batch to %s: %w", tableName, err)
		}
	}
	return nil
}

// TransformData transforms the data into Star schema
func (s *BQ) TransformData(ctx context.Context) error {
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
	cityMap := make(map[string]models.AirportCity)
	for _, c := range cities {
		cityMap[c.IATACode] = c
	}

	hotelMap := make(map[string]models.Hotel)
	for _, h := range hotels {
		hotelMap[h.HotelID] = h
	}

	reviewMap := make(map[string][]models.HotelReview)
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

		star := StarHotel{
			City:               city.Name,
			NearestAirportCode: city.IATACode,
			Latitude:           h.Latitude,
			Longitude:          h.Longitude,
			HotelName:          h.Name,
			Address:            h.StreetAddress,
			GoogleRating:       h.GoogleRating,
			AdminOverride:      "",
		}
		starHotels = append(starHotels, star)
	}

	// Insert to star_hotels table
	table := s.BQClient.Dataset(s.DatasetID).Table("star_hotels")
	uploader := table.Uploader()
	return uploader.Put(ctx, starHotels)
}

// Fetch functions
func (s *BQ) fetchCities(ctx context.Context) ([]models.AirportCity, error) {
	q := s.BQClient.Query("SELECT * FROM `" + s.DatasetID + ".cities`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	var cities []models.AirportCity
	for {
		var c models.AirportCity
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

func (s *BQ) fetchHotels(ctx context.Context) ([]models.Hotel, error) {
	q := s.BQClient.Query("SELECT * FROM `" + s.DatasetID + ".hotels`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	var hotels []models.Hotel
	for {
		var h models.Hotel
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

func (s *BQ) fetchReviews(ctx context.Context) ([]models.HotelReview, error) {
	q := s.BQClient.Query("SELECT * FROM `" + s.DatasetID + ".reviews`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	var reviews []models.HotelReview
	for {
		var r models.HotelReview
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

// AddVectorColumnToHotelReviews adds an embedding column to hotel_reviews table
func (s *BQ) AddVectorColumnToHotelReviews(ctx context.Context) error {
	query := fmt.Sprintf("ALTER TABLE `%s.%s.hotel_reviews` ADD COLUMN IF NOT EXISTS embedding ARRAY<FLOAT64>",
		s.ProjectID, s.DatasetID)
	q := s.BQClient.Query(query)
	job, err := q.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run ALTER TABLE: %w", err)
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("job failed: %w", err)
	}
	if status.Err() != nil {
		return fmt.Errorf("job error: %w", status.Err())
	}
	return nil
}

// GenerateEmbeddingsForHotelReviews populates embeddings using Vertex AI UDF
func (s *BQ) GenerateEmbeddingsForHotelReviews(ctx context.Context) error {
	query := fmt.Sprintf("UPDATE `%s.%s.hotel_reviews` r SET embedding = ML.GENERATE_EMBEDDING(MODEL `cloud-ai-ml.textembedding-gecko`, r.review_text) WHERE embedding IS NULL",
		s.ProjectID, s.DatasetID)
	q := s.BQClient.Query(query)
	job, err := q.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run UPDATE: %w", err)
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("job failed: %w", err)
	}
	if status.Err() != nil {
		return fmt.Errorf("job error: %w", status.Err())
	}
	return nil
}

// SearchSimilarReviews performs cosine similarity search on hotel_reviews
func (s *BQ) SearchSimilarReviews(ctx context.Context, queryEmbedding []float64, limit int) ([]models.HotelReview, error) {
	// Convert embedding to BigQuery array format
	embStr := "ARRAY["
	for i, v := range queryEmbedding {
		if i > 0 {
			embStr += ", "
		}
		embStr += fmt.Sprintf("%f", v)
	}
	embStr += "]"

	query := fmt.Sprintf("SELECT id, hotel_id, source, source_review_id, reviewer_name, reviewer_location, rating, review_text, review_date, verified, helpful_count, room_type, travel_type, stay_date, created_at, updated_at FROM `%s.%s.hotel_reviews` WHERE embedding IS NOT NULL ORDER BY VECTOR_DISTANCE(embedding, %s, COSINE) LIMIT %d",
		s.ProjectID, s.DatasetID, embStr, limit)

	q := s.BQClient.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	var reviews []models.HotelReview
	for {
		var r models.HotelReview
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

// cosineSimilarity utility function
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
