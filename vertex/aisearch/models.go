package aisearch

import (
	"time"
)

// City represents a city entry
type City struct {
	ID        int64     `json:"id" bigquery:"id"`
	Name      string    `json:"name" bigquery:"name"`
	Country   string    `json:"country" bigquery:"country"`
	IATACode  string    `json:"iata_code" bigquery:"iata_code"`
	Latitude  float64   `json:"latitude" bigquery:"latitude"`
	Longitude float64   `json:"longitude" bigquery:"longitude"`
	CreatedAt time.Time `json:"created_at" bigquery:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bigquery:"updated_at"`
}

// Hotel represents a hotel entry
type Hotel struct {
	ID           int64     `json:"id" bigquery:"id"`
	Name         string    `json:"name" bigquery:"name"`
	Address      string    `json:"address" bigquery:"address"`
	City         string    `json:"city" bigquery:"city"`
	IATACode     string    `json:"iata_code" bigquery:"iata_code"`
	Latitude     float64   `json:"latitude" bigquery:"latitude"`
	Longitude    float64   `json:"longitude" bigquery:"longitude"`
	GoogleRating float64   `json:"google_rating" bigquery:"google_rating"`
	CreatedAt    time.Time `json:"created_at" bigquery:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bigquery:"updated_at"`
}

// Review represents a review entry
type Review struct {
	ID        int64     `json:"id" bigquery:"id"`
	HotelID   int64     `json:"hotel_id" bigquery:"hotel_id"`
	Rating    int       `json:"rating" bigquery:"rating"`
	Quality   int       `json:"quality" bigquery:"quality"`
	Quiet     int       `json:"quiet" bigquery:"quiet"`
	Text      string    `json:"text" bigquery:"text"`
	CreatedAt time.Time `json:"created_at" bigquery:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bigquery:"updated_at"`
}

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
