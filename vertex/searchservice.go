package vertex

import (
	"context"
	"fmt"
	"log"
	"strings"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/genai" // Added for GenAI embeddings
	"google.golang.org/protobuf/types/known/structpb"
)

type SearchInput struct {
	FilterCityCountry bool
	FilterRating      bool
	Rating            int
	Continent         string
	City              string
	Country           string
}

type Config struct {
	ProjectID                    string `mapstructure:"project_id"`
	Location                     string `mapstructure:"location"`
	DatasetID                    string `mapstructure:"dataset_id"`
	IndexID                      string `mapstructure:"index_id"`
	EndpointID                   string `mapstructure:"endpoint_id"`
	DeployedIndexID              string `mapstructure:"deployed_index_id"`
	GenAIUseVertexAI             bool   `mapstructure:"google_genai_use_vertexai"`
	GoogleApplicationCredentials string `mapstructure:"google_application_credentials"`
	EndpointPublicDomainName     string `mapstructure:"endpoint_public_domain_name"`
	Limit                        int    `mapstructure:"limit"`
	Query                        string `mapstructure:"query"`
	Prompt                       string `mapstructure:"prompt"`
	CompletionModel              string `mapstructure:"completion_model"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()

	// Load YAML as base
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// OS env vars take precedence
	v.AutomaticEnv()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// VertexSearchService handles Vertex AI Vector Search operations
type VertexSearchService struct {
	// For performing real-time FindNeighbors queries against a DEPLOYED IndexEndpoint
	//indexEndpointClient *aiplatform.IndexEndpointClient
	matchClient *aiplatform.MatchClient
	genaiClient genai.Client
	projectID   string
	location    string
	datasetID   string
}

// NewVertexSearchService creates a new service
func NewVertexSearchService(ctx context.Context, config *Config) (*VertexSearchService, error) {

	clientOptions := []option.ClientOption{
		option.WithEndpoint(fmt.Sprintf("%s:443", config.EndpointPublicDomainName)), // Public domain requires port 443
	}
	matchClient, err := aiplatform.NewMatchClient(ctx, clientOptions...)
	if err != nil {
		log.Fatalf("Failed to create IndexEndpointClient: %v", err)
	}

	// Set genai client to use Vertex backend and ADC
	clientConfig := genai.ClientConfig{
		Backend:     genai.BackendVertexAI,
		Project:     config.ProjectID,
		Location:    config.Location,
		HTTPClient:  nil,
		HTTPOptions: genai.HTTPOptions{},
	}

	client, err := genai.NewClient(ctx, &clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	return &VertexSearchService{
		matchClient: matchClient,
		genaiClient: *client,
		projectID:   config.ProjectID,
		location:    config.Location,
		datasetID:   config.DatasetID,
	}, nil
}

func (s *VertexSearchService) Close() {
	err := s.matchClient.Close()
	if err != nil {
		log.Fatalf("Failed to close MatchClient: %v", err)
	}
}

func (s *VertexSearchService) PromptCompletion(ctx context.Context, config Config, question string, results []map[string]any) (string, error) {
	model := config.CompletionModel
	resultsStr := ""
	for _, r := range results {
		resultsStr += fmt.Sprintf("- Hotel: %v, City: %v, Review: %v\n", r["hotel_name"], r["city"], r["review_text"])
	}

	prompt := genai.Text(fmt.Sprintf(config.Prompt, question, resultsStr))
	resp, err := s.genaiClient.Models.GenerateContent(ctx, model, prompt, &genai.GenerateContentConfig{})

	if err != nil {
		return "", err

	}
	return resp.Text(), nil

}

// GenerateEmbedding converts a user question string into a 768-dimensional vector using Gemini embedding model
func (s *VertexSearchService) GenerateEmbedding(ctx context.Context, question string) ([]float32, error) {
	content := genai.NewContentFromText(question, "")
	result, err := s.genaiClient.Models.EmbedContent(ctx, "gemini-embedding-001", []*genai.Content{content}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated")
	}

	// Convert to []float32
	embedding := make([]float32, len(result.Embeddings[0].Values))
	for i, v := range result.Embeddings[0].Values {
		embedding[i] = float32(v)
	}
	return embedding, nil
}

// VertexSearchEndpoint performs similarity search against a deployed IndexEndpoint.
func (s *VertexSearchService) VertexSearchEndpoint(ctx context.Context, config Config, queryEmbedding []float32, params SearchInput) ([]map[string]any, error) {

	// The IndexEndpoint path format is 'projects/{project_id}/locations/{location}/indexEndpoints/{index_endpoint_id}'
	endpointPath := fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s", s.projectID, s.location, config.EndpointID)
	// Convert []float32 to []float64 for the API
	featureVector := make([]float64, len(queryEmbedding))
	for i, v := range queryEmbedding {
		featureVector[i] = float64(v)
	}

	city := params.City
	country := params.Country
	continent := params.Continent
	rating := int64(params.Rating)

	var restrictsParams []*aiplatformpb.IndexDatapoint_Restriction

	var numericRestrictsParams []*aiplatformpb.IndexDatapoint_NumericRestriction

	restrictsRating := &aiplatformpb.IndexDatapoint_NumericRestriction{
		Namespace: "rating",
		Value:     &aiplatformpb.IndexDatapoint_NumericRestriction_ValueInt{ValueInt: rating},
		Op:        aiplatformpb.IndexDatapoint_NumericRestriction_GREATER_EQUAL,
	}

	if params.FilterRating {
		numericRestrictsParams = append(numericRestrictsParams, restrictsRating)
	}

	restrictsCity := &aiplatformpb.IndexDatapoint_Restriction{
		Namespace: "city",
		AllowList: []string{city},
	}

	restrictsCountry := &aiplatformpb.IndexDatapoint_Restriction{
		Namespace: "country",
		AllowList: []string{country},
	}

	restrictsContinent := &aiplatformpb.IndexDatapoint_Restriction{
		Namespace: "continent",
		AllowList: []string{continent},
	}

	if continent != "All Regions" && continent != "" {
		restrictsParams = append(restrictsParams, restrictsContinent)
	}

	if params.FilterCityCountry {
		restrictsParams = append(restrictsParams, restrictsCity)
		restrictsParams = append(restrictsParams, restrictsCountry)
	}

	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint:   endpointPath,
		DeployedIndexId: config.DeployedIndexID, // This is the ID of the deployed index, not the endpoint ID
		// The query is an array of objects, where each object has a Datapoint

		Queries: []*aiplatformpb.FindNeighborsRequest_Query{
			{
				Datapoint: &aiplatformpb.IndexDatapoint{
					FeatureVector:    queryEmbedding,
					Restricts:        restrictsParams,
					NumericRestricts: numericRestrictsParams,
				},

				NeighborCount: int32(config.Limit),
			},
		},
		ReturnFullDatapoint: true,
	}

	resp, err := s.matchClient.FindNeighbors(ctx, req)
	if err != nil {
		log.Fatalf("Failed to find neighbors: %v", err)
	}

	var reviews []map[string]any

	for _, nearestNeighbors := range resp.GetNearestNeighbors() {
		for _, neighbor := range nearestNeighbors.GetNeighbors() {
			//fmt.Printf("  Datapoint ID: %s, Distance: %f\n", neighbor.GetDatapoint().GetDatapointId(), neighbor.GetDistance())
			searchResult := map[string]any{
				"id":       neighbor.GetDatapoint().GetDatapointId(),
				"distance": neighbor.GetDistance(),
			}
			// Access metadata
			if metadata := neighbor.GetDatapoint().GetEmbeddingMetadata(); metadata != nil {
				for key, value := range metadata.Fields {
					if key == "city" || key == "country" || key == "hotel_name" || key == "review_text" {
						switch v := value.GetKind().(type) {
						case *structpb.Value_StringValue:
							searchResult[key] = strings.TrimSpace(v.StringValue)
						case *structpb.Value_NumberValue:
							searchResult[key] = v.NumberValue
						case *structpb.Value_BoolValue:

							searchResult[key] = v.BoolValue
						default:

							searchResult[key] = value
						}
					}
				}
			}
			reviews = append(reviews, searchResult)

		}
	}
	return reviews, nil
}
