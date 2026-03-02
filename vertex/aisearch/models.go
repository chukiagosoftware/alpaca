package aisearch

import "github.com/chukiagosoftware/alpaca/internal/orm"

// for Hotel, Review structs ^^

// StarHotel represents the enriched star schema row
type StarHotel struct {
	City               string    `json:"city" bigquery:"city"`
	NearestAirportCode string    `json:"nearest_airport_code" bigquery:"nearest_airport_code"`
	Latitude           float64   `json:"latitude" bigquery:"latitude"`
	Longitude          float64   `json:"longitude" bigquery:"longitude"`
	HotelName          string    `json:"hotel_name" bigquery:"hotel_name"`
	Address            string    `json:"address" bigquery:"address"`
	GoogleRating       float64   `json:"google_rating" bigquery:"google_rating"`
	OverallRating      float64   `json:"overall_rating" bigquery:"overall_rating"`
	QualityRating      float64   `json:"quality_rating" bigquery:"quality_rating"`
	QuietRating        float64   `json:"quiet_rating" bigquery:"quiet_rating"`
	AdminOverride      string    `json:"admin_override" bigquery:"admin_override"`
	Embedding          []float32 `json:"embedding" bigquery:"embedding"` // for vector search
}

// UploadRequest represents the payload for batch upload
type UploadRequest struct {
	Data interface{} `json:"data"`
}

// VectorSearchRequest for user prompts
type VectorSearchRequest struct {
	Query string `json:"query"`
}

// VectorSearchResponse
type VectorSearchResponse struct {
	Results []StarHotel `json:"results"`
}

// LLMRequest for generating response
type LLMRequest struct {
	Query         string      `json:"query"`
	Prompt        string      `json:"prompt"`
	SearchResults []StarHotel `json:"search_results"`
}

// LLMResponse
type LLMResponse struct {
	Answer string `json:"answer"`
}
