package models

import (
	"time"
)

// HotelReview represents the hotel_reviews table (DB model with GORM, JSON, and BigQuery tags)
type HotelReview struct {
	ID               int32     `gorm:"primaryKey" json:"id" bigquery:"id"`
	HotelID          string    `gorm:"not null" json:"hotelId" bigquery:"hotel_id"`
	Source           string    `gorm:"not null;uniqueIndex:idx_source_review_id" json:"source" bigquery:"source"`
	SourceReviewID   string    `gorm:"uniqueIndex:idx_source_review_id" json:"sourceReviewId" bigquery:"source_review_id"`
	ReviewerName     string    `json:"reviewerName,omitempty" bigquery:"reviewer_name"`
	ReviewerLocation string    `json:"reviewerLocation,omitempty" bigquery:"reviewer_location"`
	Rating           float64   `json:"rating,omitempty" bigquery:"rating"`
	ReviewText       string    `gorm:"not null" json:"reviewText" bigquery:"review_text"`
	ReviewDate       time.Time `json:"reviewDate,omitempty" bigquery:"review_date"`
	Verified         bool      `gorm:"default:false" json:"verified" bigquery:"verified"`
	HelpfulCount     int       `json:"helpfulCount" bigquery:"helpful_count"`
	RoomType         string    `json:"roomType,omitempty" bigquery:"room_type"`
	TravelType       string    `json:"travelType,omitempty" bigquery:"travel_type"`
	StayDate         time.Time `json:"stayDate,omitempty" bigquery:"stay_date"`
	CreatedAt        time.Time `json:"createdAt" bigquery:"created_at"`
	UpdatedAt        time.Time `json:"updatedAt" bigquery:"updated_at"`
	GoogleMapsURI    string    `gorm:"column:google_maps_uri" json:"googleMapsURI" bigquery:"google_maps_uri"`
	Photo            string    `gorm:"column:photo_name" json:"photo_name" bigquery:"photo_name"`
	InsertID         string    `gorm:"column:insert_id" json:"insertId" bigquery:"insert_id"` // BigQuery auto deduplication
}
