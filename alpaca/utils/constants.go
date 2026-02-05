package utils

// TestHotelIDs contains the hotel IDs available in the Amadeus test environment
// These are specifically for the Hotel Ratings API (v2) testing
// Source: https://github.com/amadeus4dev/data-collection/blob/master/data/hotelratings.md
var TestHotelIDs = []string{
	// London hotels (10 total)
	"TELONMFS", // HILTON LONDON PADDINGTON
	"PILONBHG", // PREMIER INN LONDON TOLWORTH
	"RTLONWAT", // NOVOTEL LONDON WATERLOO
	"RILONJBG", // THE LANESBOROUGH
	"HOLON187", // THE NADLER SOHO
	"AELONCNP", // BATTY LANGLEY'S
	"SJLONCLR", // THE CHESTERFIELD MAYFAIR
	"DKLONDSF", // THE DORCHESTER HOTEL
	"BBLONBTL", // STAR HOTEL BED & BREAKFAST
	"CTLONCMB", // ST MARTINS LANE HOTEL
}

// TestHotelNames maps hotel IDs to their names for reference
var TestHotelNames = map[string]string{
	"TELONMFS": "HILTON LONDON PADDINGTON",
	"PILONBHG": "PREMIER INN LONDON TOLWORTH",
	"RTLONWAT": "NOVOTEL LONDON WATERLOO",
	"RILONJBG": "THE LANESBOROUGH",
	"HOLON187": "THE NADLER SOHO",
	"AELONCNP": "BATTY LANGLEY'S",
	"SJLONCLR": "THE CHESTERFIELD MAYFAIR",
	"DKLONDSF": "THE DORCHESTER HOTEL",
	"BBLONBTL": "STAR HOTEL BED & BREAKFAST",
	"CTLONCMB": "ST MARTINS LANE HOTEL",
}

// Environment variable names
const (
	HotelSearchRadius     = "HOTEL_SEARCH_RADIUS"
	HotelSearchRadiusUnit = "HOTEL_SEARCH_RADIUS_UNIT"
)

// Default values
const (
	DefaultRadius     = "100"
	DefaultRadiusUnit = "MILE"
)

// City represents a city with IATA code for hotel searches
type City struct {
	Name     string
	Country  string
	IATACode string
}
