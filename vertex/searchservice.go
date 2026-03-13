package vertex

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	vectorsearch "cloud.google.com/go/vectorsearch/apiv1"
	vectorsearchpb "cloud.google.com/go/vectorsearch/apiv1/vectorsearchpb"

	// Existing AI Platform client for MatchClient (for FindNeighbors on deployed endpoint)
	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"

	"cloud.google.com/go/bigquery"
	"github.com/chukiagosoftware/alpaca/models"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// VertexSearchService handles Vertex AI Vector Search operations
type VertexSearchService struct {
	// Single client for all vectorsearch management operations (Index, IndexEndpoint, Datapoints)
	vsClient *vectorsearch.Client
	// Separate client for real-time FindNeighbors queries against a DEPLOYED IndexEndpoint
	aiplatformMatchClient *aiplatform.MatchClient
	projectID             string
	location              string
	bq                    *BQ
}

func NewVertexSearchService(ctx context.Context, projectID, location string, bq *BQ) (*VertexSearchService, error) {
	clientOptions := []option.ClientOption{option.WithEndpoint(location + "-aiplatform.googleapis.com:443")}

	// Correct instantiation as per your example
	vsClient, err := vectorsearch.NewClient(ctx, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create vectorsearch.Client: %w", err)
	}

	aiplatformMatchClient, err := aiplatform.NewMatchClient(ctx, clientOptions...)
	if err != nil {
		vsClient.Close() // Close the first client if the second fails
		return nil, fmt.Errorf("failed to create aiplatform.MatchClient: %w", err)
	}

	return &VertexSearchService{
		vsClient:              vsClient,
		aiplatformMatchClient: aiplatformMatchClient,
		projectID:             projectID,
		location:              location,
		bq:                    bq,
	}, nil
}

// Close ensures all underlying clients are closed, releasing resources.
func (s *VertexSearchService) Close() {
	if s.vsClient != nil {
		s.vsClient.Close()
	}
	if s.aiplatformMatchClient != nil {
		s.aiplatformMatchClient.Close()
	}
}

// CreateIndex creates a new Vertex AI Vector Search Index.
// It follows the structure you provided in your example.
// See: https://pkg.go.dev/cloud.google.com/go/vectorsearch/apiv1/vectorsearchpb#CreateIndexRequest
func (s *VertexSearchService) CreateIndex(ctx context.Context, displayName, description string, dimensions int32, isBruteForce bool) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", s.projectID, s.location)

	algorithmConfig := &vectorsearchpb.Index_Metadata_Config_TreeAhConfig{} // Default to TreeAH
	if isBruteForce {
		algorithmConfig = nil // Set to nil for brute force
	}

	req := &vectorsearchpb.CreateIndexRequest{
		Parent: parent,
		Index: &vectorsearchpb.Index{
			DisplayName: displayName,
			Description: description,
			Metadata: &vectorsearchpb.Index_Metadata{
				Config: &vectorsearchpb.Index_Metadata_Config{
					Dimensions: dimensions,
					// You can specify more detailed configuration here
					// E.g., distance measure, numeric_namespaces, custom_namespaces, etc.
					// See: https://pkg.go.dev/cloud.google.com/go/vectorsearch/apiv1/vectorsearchpb#Index_Metadata_Config
					DistanceMeasureType: vectorsearchpb.Index_Metadata_Config_DOT_PRODUCT, // Common choice
					AlgorithmConfig:     &vectorsearchpb.Index_Metadata_Config_TreeAhConfig_{TreeAhConfig: algorithmConfig},
				},
			},
			// Index data must be sourced from GCS for initial build
			MetadataSchemaUri: "gs://cloud-aiplatform/schema/matching_engine/index_metadata_gcs.yaml", // Required for M.E. indexes
		},
	}

	log.Printf("Creating Index '%s' in %s...", displayName, parent)
	op, err := s.vsClient.CreateIndex(ctx, req) // Use the correct client (s.vsClient)
	if err != nil {
		return "", fmt.Errorf("failed to start CreateIndex operation: %w", err)
	}

	resp, err := op.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create Index: %w", err)
	}

	log.Printf("Index '%s' created successfully. Resource name: %s", resp.GetDisplayName(), resp.GetName())
	return resp.GetName(), nil // Returns the full resource name of the created index
}

// UploadEmbeddingsToIndex uploads embeddings from BigQuery to a Vertex AI Index in batches.
// indexName should be the full resource name of the Index, e.g., "projects/my-project/locations/us-central1/indexes/my-index-id"
func (s *VertexSearchService) UploadEmbeddingsToIndex(ctx context.Context, indexName string) error {
	tableName := fmt.Sprintf("`%s.%s.reviews`", s.bq.ProjectID, s.bq.DatasetID) // Using 'reviews' based on UI_CONTEXT
	// Make sure your BigQuery table has an 'id' (INT64) and 'embedding' (ARRAY<FLOAT64>) column.
	query := "SELECT id, embedding FROM " + tableName + " WHERE embedding IS NOT NULL"
	q := s.bq.BQClient.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read from BigQuery: %w", err)
	}

	const batchSize = 1000 // Adjust based on your embedding size and API limits

	var currentBatch []*vectorsearchpb.IndexDatapoint
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

		dp := &vectorsearchpb.IndexDatapoint{
			DatapointId:   fmt.Sprintf("%d", row.ID),
			FeatureVector: featureVector,
		}
		currentBatch = append(currentBatch, dp)
		recordCount++

		if len(currentBatch) >= batchSize {
			if err := s.upsertBatch(ctx, indexName, currentBatch); err != nil {
				return err
			}
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 {
		if err := s.upsertBatch(ctx, indexName, currentBatch); err != nil {
			return err
		}
	}

	log.Printf("Successfully uploaded %d embeddings to Vertex AI Vector Search index '%s'.", recordCount, indexName)
	return nil
}

// upsertBatch is a helper function to send a batch of datapoints to the Vertex AI Index.
func (s *VertexSearchService) upsertBatch(ctx context.Context, indexName string, datapoints []*vectorsearchpb.IndexDatapoint) error {
	req := &vectorsearchpb.UpsertDatapointsRequest{
		Index:      indexName, // Full resource name for the Index
		Datapoints: datapoints,
	}
	log.Printf("Upserting batch of %d datapoints to index '%s'...", len(datapoints), indexName)
	// Use the correct client (s.vsClient) to call UpsertDatapoints
	_, err := s.vsClient.UpsertDatapoints(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to upsert datapoints to index '%s': %w", indexName, err)
	}
	return nil
}

// VertexSearch (Query Index without deploying an endpoint)
// IMPORTANT: Vertex AI Vector Search does NOT natively offer a direct API for
// similarity search against an *undeployed* Index. Similarity search requires
// deploying the index to an IndexEndpoint.
//
// This function will clarify this limitation. For offline, non-scalable
// similarity checks without deployment, you would typically use BigQuery's
// VECTOR_DISTANCE function directly on your embeddings column.
func (s *VertexSearchService) VertexSearch(ctx context.Context, indexID string, queryEmbedding []float32, limit int) ([]models.HotelReview, error) {
	log.Printf("Attempted to query undeployed index %s for similarity search.", indexID)
	return nil, fmt.Errorf("Vertex AI Vector Search does not support similarity search directly against an undeployed index. " +
		"You must deploy the index to an IndexEndpoint for query functionality.")
	// If you needed to read *raw datapoints* (not similarity search) from the Index,
	// you might use s.vsClient.ReadIndexDatapoints, but that's different.
}

// DeployEndpoint deploys an Index to a new or existing IndexEndpoint.
// indexID: The ID of the Index to deploy.
// endpointID: The ID of an existing IndexEndpoint, or leave empty to create a new one.
// Returns the full resource name of the deployed IndexEndpoint.
func (s *VertexSearchService) DeployEndpoint(ctx context.Context, indexID string, endpointID string) (string, error) {
	indexResourceName := fmt.Sprintf("projects/%s/locations/%s/indexes/%s", s.projectID, s.location, indexID)

	// First, check if endpointID is provided; if not, create a new IndexEndpoint
	var indexEndpointResourceName string
	if endpointID == "" {
		log.Printf("Creating new IndexEndpoint for Index %s...", indexID)
		createEndpointReq := &vectorsearchpb.CreateIndexEndpointRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", s.projectID, s.location),
			IndexEndpoint: &vectorsearchpb.IndexEndpoint{
				DisplayName: fmt.Sprintf("%s-endpoint-%d", indexID, time.Now().Unix()),
				Network:     fmt.Sprintf("projects/%s/global/networks/default", s.projectID), // Or your specific network
			},
		}
		createOp, err := s.vsClient.CreateIndexEndpoint(ctx, createEndpointReq)
		if err != nil {
			return "", fmt.Errorf("failed to start create IndexEndpoint operation: %w", err)
		}
		createdEndpoint, err := createOp.Wait(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to create IndexEndpoint: %w", err)
		}
		indexEndpointResourceName = createdEndpoint.GetName()
		log.Printf("Created IndexEndpoint: %s", indexEndpointResourceName)
	} else {
		indexEndpointResourceName = fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s", s.projectID, s.location, endpointID)
		log.Printf("Using existing IndexEndpoint: %s", indexEndpointResourceName)
	}

	log.Printf("Deploying Index %s to IndexEndpoint %s...", indexResourceName, indexEndpointResourceName)
	deployReq := &vectorsearchpb.DeployIndexRequest{
		IndexEndpoint: indexEndpointResourceName,
		DeployedIndex: &vectorsearchpb.DeployedIndex{
			Id:    indexID + "_deployed", // A unique ID for the deployed index within the endpoint
			Index: indexResourceName,
			// You might need to specify a machine_type for the serving nodes if not using default
			// DedicatedResources: &vectorsearchpb.DedicatedResources{
			// 	MachineSpec: &vectorsearchpb.MachineSpec{
			// 		MachineType: "e2-standard-2",
			// 	},
			// 	MinReplicaCount: 1,
			// },
		},
	}

	deployOp, err := s.vsClient.DeployIndex(ctx, deployReq)
	if err != nil {
		return "", fmt.Errorf("failed to start deploy Index operation: %w", err)
	}
	_, err = deployOp.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to deploy Index: %w", err)
	}

	log.Printf("Index %s successfully deployed to endpoint %s.", indexID, indexEndpointResourceName)
	return indexEndpointResourceName, nil
}

// VertexSearchEndpoint performs similarity search against a DEPLOYED Vertex AI IndexEndpoint.
// indexEndpointResourceName: The full resource name of the deployed Vertex AI IndexEndpoint.
//
//	e.g., "projects/my-project/locations/us-central1/indexEndpoints/my-endpoint-id"
//
// queryEmbedding: The embedding of the query for similarity search (expected []float32).
func (s *VertexSearchService) VertexSearchEndpoint(ctx context.Context, indexEndpointResourceName string, queryEmbedding []float32, limit int) ([]models.HotelReview, error) {
	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint: indexEndpointResourceName, // Full resource name for the IndexEndpoint
		Queries: []*aiplatformpb.FindNeighborsRequest_Query{
			{
				Datapoint: &aiplatformpb.IndexDatapoint{
					FeatureVector: queryEmbedding,
				},
				NeighborCount: int32(limit),
			},
		},
	}
	// Use the aiplatformMatchClient for FindNeighbors
	resp, err := s.aiplatformMatchClient.FindNeighbors(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to find neighbors using endpoint '%s' with MatchClient: %w", indexEndpointResourceName, err)
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
	tableName := fmt.Sprintf("`%s.%s.reviews`", s.bq.ProjectID, s.bq.DatasetID)

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

	reviewID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid review ID format: %w", err)
	}

	q.Parameters = []bigquery.QueryParameter{
		{Name: "review_id", Value: reviewID},
	}

	it, err := q.Read(ctx)
	if err != iterator.Done && err != nil {
		return nil, fmt.Errorf("failed to read review from BigQuery for ID %s: %w", id, err)
	}
	var review models.HotelReview
	if err == iterator.Done {
		return nil, fmt.Errorf("review with ID %s not found", id)
	}
	err = it.Next(&review)
	if err == iterator.Done {
		return nil, fmt.Errorf("review with ID %s not found after read", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read review row from BigQuery for ID %s: %w", id, err)
	}
	return &review, nil
}
