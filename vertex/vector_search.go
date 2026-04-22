package vertex

import (
	"context"
	"fmt"

	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/genai"
)

type SearchForm struct {
	Question    string `form:"question"`
	Continent   string `form:"continent"`
	CityCountry string `form:"citycountry"`
	Rating      string `form:"rating"`
	LLMChoice   string `form:"llm"`
}

type SearchInput struct {
	Question          string
	Continent         string
	City              string
	Country           string
	Rating            int
	FilterRating      bool
	FilterCityCountry bool
	PreferredModel    string
}

type VectorResult struct {
	ID       string  `json:"id"`
	Distance float64 `json:"distance"`
}

func float32Ptr(v float32) *float32 {
	return &v
}

func (s *VertexSearchService) GenerateEmbedding(ctx context.Context, question string) ([]float32, error) {
	content := genai.NewContentFromText(question, "")
	result, err := s.genaiClient.Models.EmbedContent(ctx, "gemini-embedding-001", []*genai.Content{content}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated")
	}

	embedding := make([]float32, len(result.Embeddings[0].Values))
	for i, v := range result.Embeddings[0].Values {
		embedding[i] = float32(v)
	}
	return embedding, nil
}

func (s *VertexSearchService) VertexSearchEndpoint(ctx context.Context, config Config, queryEmbedding []float32, params SearchInput) ([]VectorResult, error) {
	endpointPath := fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s", s.projectID, s.location, config.EndpointID)

	city := params.City
	country := params.Country
	continent := params.Continent
	rating := int64(params.Rating)

	var restrictsParams []*aiplatformpb.IndexDatapoint_Restriction
	var numericRestrictsParams []*aiplatformpb.IndexDatapoint_NumericRestriction

	if params.FilterRating {
		numericRestrictsParams = append(numericRestrictsParams, &aiplatformpb.IndexDatapoint_NumericRestriction{
			Namespace: "rating",
			Value:     &aiplatformpb.IndexDatapoint_NumericRestriction_ValueInt{ValueInt: rating},
			Op:        aiplatformpb.IndexDatapoint_NumericRestriction_GREATER_EQUAL,
		})
	}

	if continent != "" {
		restrictsParams = append(restrictsParams, &aiplatformpb.IndexDatapoint_Restriction{
			Namespace: "continent",
			AllowList: []string{continent},
		})
	}

	if params.FilterCityCountry {
		restrictsParams = append(restrictsParams, &aiplatformpb.IndexDatapoint_Restriction{
			Namespace: "city",
			AllowList: []string{city},
		})
		restrictsParams = append(restrictsParams, &aiplatformpb.IndexDatapoint_Restriction{
			Namespace: "country",
			AllowList: []string{country},
		})
	}

	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint:   endpointPath,
		DeployedIndexId: config.DeployedIndexID,
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
		ReturnFullDatapoint: false,
	}

	resp, err := s.matchClient.FindNeighbors(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to find neighbors: %w", err)
	}

	var results []VectorResult
	for _, nearestNeighbors := range resp.GetNearestNeighbors() {
		for _, neighbor := range nearestNeighbors.GetNeighbors() {
			results = append(results, VectorResult{
				ID:       neighbor.GetDatapoint().GetDatapointId(),
				Distance: neighbor.GetDistance(),
			})
		}
	}

	return results, nil
}
