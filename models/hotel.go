package models

import (
	"time"
)

type Hotel struct {
	ID                uint   `gorm:"primaryKey"`
	HotelID           string `gorm:"uniqueIndex;not null"`
	Source            string `gorm:"not null"`
	SourceHotelID     string
	Name              string `gorm:"not null"`
	City              string
	Country           string
	Latitude          *float64
	Longitude         *float64
	StreetAddress     string `gorm:"column:street_address"`
	PostalCode        string `gorm:"column:postal_code"`
	Phone             string
	Website           string
	Email             string
	AmadeusRating     float64 `gorm:"column:amadeus_rating"`
	ExpediaRating     float64 `gorm:"column:expedia_rating"`
	TripadvisorRating float64 `gorm:"column:tripadvisor_rating"`
	GoogleRating      float64 `gorm:"column:google_rating"`
	BookingRating     float64 `gorm:"column:booking_rating"`
	YelpRating        float64 `gorm:"column:yelp_rating"`
	Recommended       bool    `gorm:"default:false"`
	AdminFlag         bool    `gorm:"column:admin_flag;default:false"`
	Quality           bool    `gorm:"default:false"`
	Quiet             bool    `gorm:"default:false"`
	ImportantNote     string  `gorm:"column:important_note"`
	Type              string
	ChainCode         string `gorm:"column:chain_code"`
	DupeID            int64  `gorm:"column:dupe_id"`
	IATACode          string `gorm:"column:iata_code"`
	AddressJSON       string `gorm:"column:address_json"`
	GeoCodeJSON       string `gorm:"column:geo_code_json"`
	DistanceJSON      string `gorm:"column:distance_json"`
	LastUpdate        string `gorm:"column:last_update"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	StateCode         string  `gorm:"column:state_code"`
	NumberOfReviews   int     `gorm:"column:number_of_reviews;default:0"`
	NumberOfRatings   int     `gorm:"column:number_of_ratings;default:0"`
	OverallRating     float64 `gorm:"column:overall_rating"`
	Sentiments        string  `gorm:"column:sentiments"`
}
