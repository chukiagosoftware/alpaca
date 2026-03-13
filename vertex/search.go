package vertex

import (
	"context"
	"fmt"
	"log"
	"strconv"
	// "time" for backoff implementation

	"cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"cloud.google.com/go/bigquery" // Import bigquery for parameters and NullFloat64
	"github.com/chukiagosoftware/alpaca/models"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// VertexSearchService handles Vertex AI Search operations
type VertexSearchService struct {
	indexClient         *aiplatform.IndexClient         // For managing Index resources (e.g., UpsertDatapoints)
	indexEndpointClient *aiplatform.IndexEndpointClient // For managing IndexEndpoint resources (e.g., DeployIndex)
	matchClient         *aiplatform.MatchClient         // For performing real-time FindNeighbors queries
	projectID           string
	location            string
	bq                  *BQ // Reference to your BigQuery client
}

// NewVertexSearchService creates a new service
func NewVertexSearchService(ctx context.Context, projectID, location string, bq *BQ) (*VertexSearchService, error) {
	// Common endpoint for AI Platform services
	clientOptions := []option.ClientOption{option.WithEndpoint(location + "-aiplatform.googleapis.com:443")}

	indexClient, err := aiplatform.NewIndexClient(ctx, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create IndexClient: %w", err)
	}

	indexEndpointClient, err := aiplatform.NewIndexEndpointClient(ctx, clientOptions...)
	if err != nil {
		indexClient.Close()
		return nil, fmt.Errorf("failed to create IndexEndpointClient: %w", err)
	}

	// Initialize MatchClient for FindNeighbors operations
	matchClient, err := aiplatform.NewMatchClient(ctx, clientOptions...)
	if err != nil {
		indexClient.Close()
		indexEndpointClient.Close()
		return nil, fmt.Errorf("failed to create MatchClient: %w", err)
	}

	return &VertexSearchService{
		indexClient:         indexClient,
		indexEndpointClient: indexEndpointClient,
		matchClient:         matchClient, // Assign the new MatchClient
		projectID:           projectID,
		location:            location,
		bq:                  bq,
	}, nil
}

// Close ensures all underlying clients are closed, releasing resources.
func (s *VertexSearchService) Close() {
	if s.indexClient != nil {
		s.indexClient.Close()
	}
	if s.indexEndpointClient != nil {
		s.indexEndpointClient.Close()
	}
	if s.matchClient != nil { // Close the new MatchClient
		s.matchClient.Close()
	}
}

// UploadEmbeddingsToVertexSearch uploads embeddings from BigQuery to a Vertex AI Index in batches.
// indexID: The ID of the Vertex AI Index resource (not the endpoint ID).
func (s *VertexSearchService) UploadEmbeddingsToVertexSearch(ctx context.Context, indexID string) error {
	// Ensure the BigQuery table 'hotel_reviews' exists and has 'id' (INT64) and 'embedding' (ARRAY<FLOAT64>) columns.
	tableName := fmt.Sprintf("`%s.%s.reviews`", s.bq.ProjectID, s.bq.DatasetID) // Using 'reviews' based on UI_CONTEXT
	query := "SELECT id, embedding FROM " + tableName + " WHERE embedding IS NOT NULL"
	q := s.bq.BQClient.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read from BigQuery: %w", err)
	}

	const batchSize = 1000 // Adjust based on your embedding size and API limits

	var currentBatch []*aiplatformpb.IndexDatapoint
	recordCount := 0

	for {
		var row struct {
			ID        int64                  `bigquery:"id"`
			Embedding []bigquery.NullFloat64 `bigquery:"embedding"`
		}
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate BigQuery results: %w", err)
		}

		var featureVector []float32
		if len(row.Embedding) > 0 {
			featureVector = make([]float32, len(row.Embedding))
			for i, val := range row.Embedding {
				if val.Valid {
					featureVector[i] = float32(val.Float64)
				} else {
					featureVector[i] = 0.0
					log.Printf("Warning: Null embedding element found for ID %d at index %d. Setting to 0.0", row.ID, i)
				}
			}
		} else {
			log.Printf("Warning: Empty embedding found for ID %d. Skipping datapoint.", row.ID)
			continue
		}

		dp := &aiplatformpb.IndexDatapoint{
			DatapointId:   fmt.Sprintf("%d", row.ID),
			FeatureVector: featureVector,
		}
		currentBatch = append(currentBatch, dp)
		recordCount++

		if len(currentBatch) >= batchSize {
			if err := s.upsertBatch(ctx, indexID, currentBatch); err != nil {
				return err
			}
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 {
		if err := s.upsertBatch(ctx, indexID, currentBatch); err != nil {
			return err
		}
	}

	log.Printf("Successfully uploaded %d embeddings to Vertex Search index '%s'.", recordCount, indexID)
	return nil
}

// upsertBatch is a helper function to send a batch of datapoints to the Vertex AI Index.
func (s *VertexSearchService) upsertBatch(ctx context.Context, indexID string, datapoints []*aiplatformpb.IndexDatapoint) error {
	req := &aiplatformpb.UpsertDatapointsRequest{
		Index:      fmt.Sprintf("projects/%s/locations/%s/indexes/%s", s.projectID, s.location, indexID),
		Datapoints: datapoints,
	}
	log.Printf("Upserting batch of %d datapoints to index '%s'...", len(datapoints), indexID)
	_, err := s.indexClient.UpsertDatapoints(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to upsert datapoints to index '%s': %w", indexID, err)
	}
	return nil
}

// SearchVertex performs similarity search against a deployed Vertex AI IndexEndpoint using the MatchClient.
// indexEndpointID: The ID of the deployed Vertex AI IndexEndpoint resource.
// queryEmbedding: The embedding of the query for similarity search (expected []float32).
func (s *VertexSearchService) SearchVertex(ctx context.Context, indexEndpointID string, queryEmbedding []float32, limit int) ([]models.HotelReview, error) {
	// The IndexEndpoint path format is 'projects/{project_id}/locations/{location}/indexEndpoints/{index_endpoint_id}'
	endpointPath := fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s", s.projectID, s.location, indexEndpointID)

	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint: endpointPath,
		Queries: []*aiplatformpb.FindNeighborsRequest_Query{
			{
				Datapoint: &aiplatformpb.IndexDatapoint{
					FeatureVector: queryEmbedding,
				},
				NeighborCount: int32(limit),
			},
		},
	}
	// *** CRITICAL CHANGE: Use s.matchClient for FindNeighbors ***
	resp, err := s.matchClient.FindNeighbors(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to find neighbors using endpoint '%s' with MatchClient: %w", indexEndpointID, err)
	}

	var reviews []models.HotelReview
	if len(resp.NearestNeighbors) == 0 || len(resp.NearestNeighbors[0].Neighbors) == 0 {
		log.Println("No nearest neighbors found.")
		return reviews, nil
	}

	for _, neighbor := range resp.NearestNeighbors[0].Neighbors {
		review, err := s.fetchReviewByID(ctx, neighbor.Datapoint.DatapointId)
		if err != nil {
			log.Printf("Failed to fetch review %s: %v", neighbor.Datapoint.DatapointId, err)
			continue
		}
		reviews = append(reviews, *review)
	}
	return reviews, nil
}

// fetchReviewByID retrieves a single hotel review from BigQuery by its ID.
func (s *VertexSearchService) fetchReviewByID(ctx context.Context, id string) (*models.HotelReview, error) {
	tableName := fmt.Sprintf("`%s.%s.reviews`", s.bq.ProjectID, s.bq.DatasetID) // Using 'reviews' based on UI_CONTEXT

	query := fmt.Sprintf(`
		SELECT
			id,
			hotel_id,
			source,
			source_review_id,
			reviewer_name,
			reviewer_location,
			rating,
			review_text,
			review_date,
			verified,
			helpful_count,
			room_type,
			travel_type,
			stay_date,
			created_at,
			updated_at
		FROM %s
		WHERE id = @review_id`, tableName)
	q := s.bq.BQClient.Query(query)

	reviewID, err := strconv.ParseInt(id, 10, 64) // Corrected to use strconv.ParseInt
	if err != nil {
		return nil, fmt.Errorf("invalid review ID format: %w", err)
	}

	q.Parameters = []bigquery.QueryParameter{
		{Name: "review_id", Value: reviewID},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read review from BigQuery for ID %s: %w", id, err)
	}
	var review models.HotelReview
	err = it.Next(&review)
	if err == iterator.Done {
		return nil, fmt.Errorf("review with ID %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read next review from BigQuery for ID %s: %w", id, err)
	}
	return &review, nil
}
