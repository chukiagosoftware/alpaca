package vertex

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

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
func NewBigQueryService(ctx context.Context, config Config) (*BQ, error) {
	bqClient, err := bigquery.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, err
	}
	return &BQ{
		BQClient:  bqClient,
		ProjectID: config.ProjectID,
		DatasetID: config.DatasetID,
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

func UploadBatches[T any](ctx context.Context, s *BQ, tableName string, data []T) error {
	if len(data) == 0 {
		log.Println("⚠️ Empty data, skipping")
		return nil
	}

	table := s.BQClient.Dataset(s.DatasetID).Table(tableName)

	const batchSize = 5000 // 👈 Safer for big reviews (tune up if JSON est. <8MB)
	const numWorkers = 5

	batchChan := make(chan []T, numWorkers*2)
	errChan := make(chan error, numWorkers) // Buffered, no close needed
	var wg sync.WaitGroup

	// Workers: NO defer close(errChan)!
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inserter := table.Inserter()
			for batch := range batchChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// Worker loop:
				log.Printf("🔄 Worker starting batch %d rows", len(batch))
				start := time.Now()
				if err := inserter.Put(ctx, batch); err != nil {
					log.Printf("❌ Put failed (%v): %v", time.Since(start), err)
					errChan <- err
					return
				}
				log.Printf("✅ Put success %d rows in %v", len(batch), time.Since(start))
			}
		}()
	}

	// Producer
	totalBatches := (len(data) + batchSize - 1) / batchSize

	for i := 0; i < len(data); i += batchSize {
		select {
		case <-ctx.Done():
			close(batchChan)
			wg.Wait()
			return ctx.Err()
		default:
		}
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		batchChan <- data[i:end]

		if i/batchSize%5 == 0 {
			log.Printf("📤 %s: batch %d/%d (~%d rows)", tableName, i/batchSize+1, totalBatches, len(data))
		}
	}
	close(batchChan)

	wg.Wait()

	// Drain first err (safe, no panic)
	select {
	case err := <-errChan:
		return err
	default:
		// Check for more? Optional buffered drain
	}
	log.Printf("✅ %s: %d rows / %d batches", tableName, len(data), totalBatches)
	return nil
}

// Old UploadData uploads data to BigQuery in small batches
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

// Fetch functions for deprecatd AirportCity
func (bq *BQ) fetchCities(ctx context.Context) ([]models.AirportCity, error) {
	tableString := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, "airportCity")
	query := "SELECT * FROM" + " " + tableString
	q := bq.BQClient.Query(query)
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
	tableString := fmt.Sprintf("%s.%s.%s", s.ProjectID, s.DatasetID, "hotels")
	query := "SELECT * FROM" + " " + tableString
	q := s.BQClient.Query(query)
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

func (bq *BQ) fetchReviews(ctx context.Context) ([]models.HotelReview, error) {
	tableString := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, "reviews")
	queryString := "SELECT * FROM" + " " + tableString
	q := bq.BQClient.Query(queryString)
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

// fetchReviewByID fetches a hotel review by its ID from BigQuery
func (bq *BQ) fetchReviewByID(ctx context.Context, idStr string) (*models.HotelReview, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}

	tableString := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, "reviews")

	query := "SELECT * FROM" + " " + tableString + " WHERE id = @id"
	params := []bigquery.QueryParameter{
		{Name: "id", Value: id},
	}

	iter, err := bq.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var review models.HotelReview
	err = iter.Next(&review)
	if err == iterator.Done {
		return nil, fmt.Errorf("review not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read row: %w", err)
	}

	return &review, nil
}

// ExecuteQuery usage: it, err := s.ExecuteQuery(ctx, sql, params); then it.Next(&row)
func (bq *BQ) ExecuteQuery(ctx context.Context, query string, params []bigquery.QueryParameter) (*bigquery.RowIterator, error) {
	q := bq.BQClient.Query(query)
	q.Parameters = params

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}
	return it, nil
}

// GetDistinctLocations returns sorted distinct continents/countries/cities from bigReviews_embeddings.
func (bq *BQ) GetDistinctLocations(ctx context.Context) ([]LocationGroup, error) {
	richEmbeddedReviews := fmt.Sprintf("%s.%s.bigReview_embeddings", bq.ProjectID, bq.DatasetID)

	// cannot use alias defined in select in the where clause but can define it for output labelling
	sql := fmt.Sprintf(`
			SELECT continent,
    				ARRAY_AGG(DISTINCT CONCAT(city, ', ', country) ORDER BY CONCAT(city, ', ', country)) as city_countries
			FROM %s
			WHERE continent IS NOT NULL 
			  AND continent != ''
			  AND city IS NOT NULL 
			  AND city != ''
			  AND country IS NOT NULL 
			  AND country != ''
			GROUP BY continent
			ORDER BY continent`,
		richEmbeddedReviews)

	params := []bigquery.QueryParameter{
		//{
		//	Name: "continents",
		//	Value: []string{
		//		"USA",
		//		"mexico",
		//		"canada",
		//		"caribbean",
		//		"centralAmerica",
		//		"southamerica",
		//		"oceania",
		//		"europe",
		//		"asia",
		//		"africa",
		//	},
		//},
	}

	it, err := bq.ExecuteQuery(ctx, sql, params)
	if err != nil {
		return nil, err
	}
	//defer it.Close()

	var groups []LocationGroup
	for {
		var g LocationGroup
		err := it.Next(&g)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}
