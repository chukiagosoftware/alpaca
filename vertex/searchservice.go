package vertex

import (
	"context"
	"fmt"
	"log"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"google.golang.org/api/option"
	"google.golang.org/genai"
)

type VertexSearchService struct {
	matchClient      *aiplatform.MatchClient
	genaiClient      genai.Client
	completionRouter *CompletionRouter
	projectID        string
	location         string
	datasetID        string
}

func NewVertexSearchService(ctx context.Context, config *Config) (*VertexSearchService, error) {
	clientOptions := []option.ClientOption{
		option.WithEndpoint(fmt.Sprintf("%s:443", config.EndpointPublicDomainName)),
	}
	matchClient, err := aiplatform.NewMatchClient(ctx, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MatchClient: %w", err)
	}

	clientConfig := genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  config.ProjectID,
		Location: config.Location,
	}

	client, err := genai.NewClient(ctx, &clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	router, err := NewCompletionRouter(config, *client)
	if err != nil {
		return nil, err
	}

	return &VertexSearchService{
		matchClient:      matchClient,
		genaiClient:      *client,
		completionRouter: router,
		projectID:        config.ProjectID,
		location:         config.Location,
		datasetID:        config.DatasetID,
	}, nil
}

func (s *VertexSearchService) Close() {
	if err := s.matchClient.Close(); err != nil {
		log.Printf("Failed to close MatchClient: %v", err)
	}
}

func (s *VertexSearchService) CheckQuerySafety(ctx context.Context, input SearchInput) (bool, error) {
	return s.completionRouter.CheckQuerySafety(ctx, input)
}

func (s *VertexSearchService) PromptCompletion(ctx context.Context, input SearchInput, results []map[string]any) (CompletionResult, error) {
	return s.completionRouter.PromptCompletion(ctx, input, results)
}
