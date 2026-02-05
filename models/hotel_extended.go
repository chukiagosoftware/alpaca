package models

import (
	"time"
)

// Hotel represents a consolidated hotel from multiple sources
type Hotel struct {
	ID            int64    `json:"id"`
	HotelID       string   `json:"hotelId"`
	Source        string   `json:"source"` // 'amadeus', 'expedia', 'tripadvisor', 'google', etc.
	SourceHotelID string   `json:"sourceHotelId"`
	Name          string   `json:"name"`
	City          string   `json:"city"`
	Country       string   `json:"country"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	StreetAddress string   `json:"streetAddress,omitempty"`
	PostalCode    string   `json:"postalCode,omitempty"`
	Phone         string   `json:"phone,omitempty"`
	Website       string   `json:"website,omitempty"`
	Email         string   `json:"email,omitempty"`

	// Ratings from different sources
	AmadeusRating     *float64 `json:"amadeusRating,omitempty"`
	ExpediaRating     *float64 `json:"expediaRating,omitempty"`
	TripadvisorRating *float64 `json:"tripadvisorRating,omitempty"`
	GoogleRating      *float64 `json:"googleRating,omitempty"`
	BookingRating     *float64 `json:"bookingRating,omitempty"`

	// Recommendation fields
	Recommended   bool   `json:"recommended"`
	AdminFlag     bool   `json:"adminFlag"` // true = disabled by admin
	Quality       bool   `json:"quality"`
	Quiet         bool   `json:"quiet"`
	ImportantNote string `json:"importantNote,omitempty"`

	// Original fields (for backward compatibility)
	Type      string `json:"type,omitempty"`
	ChainCode string `json:"chainCode,omitempty"`
	DupeID    int64  `json:"dupeId,omitempty"`
	IATACode  string `json:"iataCode,omitempty"`

	LastUpdate string    `json:"lastUpdate,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// HotelReview represents a review from any source
type HotelReview struct {
	ID               int64      `json:"id"`
	HotelID          string     `json:"hotelId"`
	Source           string     `json:"source"` // 'tripadvisor', 'google', 'expedia', 'booking', 'hotel_website', etc.
	SourceReviewID   string     `json:"sourceReviewId"`
	ReviewerName     string     `json:"reviewerName,omitempty"`
	ReviewerLocation string     `json:"reviewerLocation,omitempty"`
	Rating           *float64   `json:"rating,omitempty"`
	ReviewText       string     `json:"reviewText"`
	ReviewDate       *time.Time `json:"reviewDate,omitempty"`
	Verified         bool       `json:"verified"`
	HelpfulCount     int        `json:"helpfulCount"`
	RoomType         string     `json:"roomType,omitempty"`
	TravelType       string     `json:"travelType,omitempty"` // 'business', 'leisure', 'family', etc.
	StayDate         *time.Time `json:"stayDate,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// HotelRecommendation represents LLM-processed recommendation
type HotelRecommendation struct {
	ID                    int64     `json:"id"`
	HotelID               string    `json:"hotelId"`
	QualityScore          float64   `json:"qualityScore"`      // 0.0 to 1.0
	QualityConfidence     float64   `json:"qualityConfidence"` // 0.0 to 1.0
	QualityReasoning      string    `json:"qualityReasoning"`
	QuietScore            float64   `json:"quietScore"`      // 0.0 to 1.0
	QuietConfidence       float64   `json:"quietConfidence"` // 0.0 to 1.0
	QuietReasoning        string    `json:"quietReasoning"`
	OverallRecommended    bool      `json:"overallRecommended"`
	RecommendationSummary string    `json:"recommendationSummary"`
	ReviewsAnalyzed       int       `json:"reviewsAnalyzed"`
	LLMModel              string    `json:"llmModel"` // 'gpt-4', 'claude', 'grok', etc.
	ProcessedAt           time.Time `json:"processedAt"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

// ReviewSource represents different review sources
const (
	SourceTripadvisor  = "tripadvisor"
	SourceGoogle       = "google"
	SourceExpedia      = "expedia"
	SourceBooking      = "booking"
	SourceHotelWebsite = "hotel_website"
	SourceBing         = "bing"
	SourceYelp         = "yelp"
	SourceAmadeus      = "amadeus"
)

// HotelSource represents different hotel data sources
const (
	HotelSourceAmadeus     = "amadeus"
	HotelSourceExpedia     = "expedia"
	HotelSourceTripadvisor = "tripadvisor"
	HotelSourceGoogle      = "google"
	HotelSourceBooking     = "booking"
)
