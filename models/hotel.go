package models

import (
	"time"
)

type Hotel struct {
	ID              int64  `gorm:"primaryKey"`
	HotelID         string `gorm:"uniqueIndex;not null" bigquery:"hotel_id"`
	Source          string `gorm:"not null" bigquery:"source"`
	SourceHotelID   string `bigquery:"source_hotel_id"`
	Name            string `gorm:"not null" bigquery:"name"`
	City            string
	Country         string
	Latitude        float64
	Longitude       float64
	StreetAddress   string `gorm:"column:street_address" bigquery:"street_address"`
	PostalCode      string `gorm:"column:postal_code" bigquery:"postal_code"`
	Phone           string
	Website         string
	Email           string
	AmadeusRating   float64 `gorm:"column:amadeus_rating" bigquery:"amadeus_rating"`
	GoogleRating    float64 `gorm:"column:google_rating" bigquery:"google_rating"`
	Recommended     bool    `gorm:"default:false"`
	AdminFlag       bool    `gorm:"column:admin_flag;default:false" bigquery:"admin_flag"`
	Quality         bool    `gorm:"default:false"`
	Quiet           bool    `gorm:"default:false"`
	ImportantNote   string  `gorm:"column:important_note" bigquery:"important_note"`
	Type            string
	DupeID          int64  `gorm:"column:dupe_id"`
	IATACode        string `gorm:"column:iata_code" bigquery:"iata_code"`
	AddressJSON     string `gorm:"column:address_json" bigquery:"address_json"`
	GeoCodeJSON     string `gorm:"column:geo_code_json" bigquery:"geo_code_json"`
	DistanceJSON    string `gorm:"column:distance_json" bigquery:"distance_json"`
	LastUpdate      string `gorm:"column:last_update"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	StateCode       string  `gorm:"column:state_code" bigquery:"state_code"`
	NumberOfReviews int     `gorm:"column:number_of_reviews;default:0" bigquery:"number_of_reviews"`
	NumberOfRatings int     `gorm:"column:number_of_ratings;default:0" bigquery:"number_of_ratings"`
	OverallRating   float64 `gorm:"column:overall_rating" bigquery:"overall_rating"`
	Sentiments      string  `gorm:"column:sentiments"`
}
