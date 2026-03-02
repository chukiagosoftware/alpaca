package models

// Hotel source constants
const (
	HotelSourceAmadeus     = "amadeus"
	HotelSourceExpedia     = "expedia"
	HotelSourceTripadvisor = "tripadvisor"
	HotelSourceGoogle      = "google"
	HotelSourceBooking     = "booking"
	HotelSourceYelp        = "yelp"
)

// AirportCity represents the airport_cities table
type AirportCity struct {
	Name         string `gorm:"primaryKey"`
	Country      string `gorm:"primaryKey"`
	IATACode     string `gorm:"column:iata_code;unique;not null"`
	AirportCount int    `gorm:"column:airport_count;default:0"`
}
