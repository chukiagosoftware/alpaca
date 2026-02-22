package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/chukiagosoftware/alpaca/models"
)

// create inserts a new hotel
func (s *hotelStorage) create(ctx context.Context, hotel *models.HotelAPIItem) error {
	// Parse address_json to extract city, country, etc.
	var addressData struct {
		CityName    string   `json:"cityName"`
		CountryCode string   `json:"countryCode"`
		Line1       []string `json:"lines"`
		PostalCode  string   `json:"postalCode"`
		StateCode   string   `json:"stateCode"`
		// Add other fields as needed
	}
	if err := json.Unmarshal(hotel.Address, &addressData); err != nil {
		log.Printf("Warning: Failed to parse address_json for hotel %s: %v", hotel.HotelID, err)
		// Continue without parsed data
	}

	query := `
		INSERT INTO hotels (hotel_id, type, chain_code, dupe_id, name, iata_code, city, country, street_address, postal_code, state_code, address_json, geo_code_json, distance_json, last_update, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hotel_id) DO UPDATE SET
			type = excluded.type,
			chain_code = excluded.chain_code,
			dupe_id = excluded.dupe_id,
			name = excluded.name,
			iata_code = excluded.iata_code,
			city = excluded.city,
			country = excluded.country,
			street_address = excluded.street_address,
			postal_code = excluded.postal_code,
		    state_code = excluded.state_code,
			address_json = excluded.address_json,
			geo_code_json = excluded.geo_code_json,
			distance_json = excluded.distance_json,
			last_update = excluded.last_update,
			updated_at = CURRENT_TIMESTAMP
	`

	addressJSON, _ := json.Marshal(hotel.Address)
	geoCodeJSON, _ := json.Marshal(hotel.GeoCode)
	distanceJSON, _ := json.Marshal(hotel.Distance)

	_, err := s.db.ExecContext(ctx, query,
		hotel.HotelID,
		hotel.Type,
		hotel.ChainCode,
		hotel.DupeID,
		hotel.Name,
		hotel.IATACode,
		addressData.CityName,                  // Populated from parsed JSON
		addressData.CountryCode,               // Populated from parsed JSON
		strings.Join(addressData.Line1, ", "), // Populated from parsed JSON,       // Populated from parsed JSON
		addressData.PostalCode,                // Populated from parsed JSON
		addressData.StateCode,                 // Populated from parsed JSON
		string(addressJSON),
		string(geoCodeJSON),
		string(distanceJSON),
		hotel.LastUpdate,
		models.HotelSourceAmadeus,
	)
	return err
}

// getHotelIDs returns all hotel IDs for processing
func (s *hotelStorage) getHotelIDs(ctx context.Context) ([]string, error) {
	query := `SELECT hotel_id FROM hotels`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hotelIDs []string
	for rows.Next() {
		var hotelID string
		if err := rows.Scan(&hotelID); err != nil {
			return nil, err
		}
		hotelIDs = append(hotelIDs, hotelID)
	}
	return hotelIDs, rows.Err()
}
