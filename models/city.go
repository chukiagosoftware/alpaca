package models

// AirportCity represents the airport_cities table
type AirportCity struct {
	IATACode     string `gorm:"primaryKey;column:iata_code;unique;not null" bigquery:"iata_code"`
	Name         string `gorm:"not null" bigquery:"name"`
	Country      string `gorm:"not null" bigquery:"country"`
	AirportCount int    `gorm:"column:airport_count;default:0" bigquery:"airport_count"`
	CountryCodes string `gorm:"column:country_codes"x bigquery:"country_codes"`
}

type Country struct {
	Name  string
	ISO2  string
	Codes string
}

// Airport represents an airport from the OpenFlights dataset
type Airport struct {
	ID       string
	Name     string
	City     string
	Country  string
	IATA     string
	ICAO     string
	Lat      string
	Lon      string
	Altitude string
	Timezone string
	DST      string
	TzDB     string
	Type     string
	Source   string
}
