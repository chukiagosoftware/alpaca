package hotelstorage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/chukiagosoftware/alpaca/database"
	"github.com/chukiagosoftware/alpaca/models"
)

type Storage struct {
	db *database.DB
}

func NewStorage(db *database.DB) *Storage {
	return &Storage{db: db}
}

// GetAllHotels retrieves all hotels from the database
func (s *Storage) GetAllHotels(ctx context.Context) ([]*models.Hotel, error) {
	query := `
		SELECT id, hotel_id, source, source_hotel_id, name, city, country,
		       latitude, longitude, street_address, postal_code, state_code, phone, website, email,
		       amadeus_rating, expedia_rating, tripadvisor_rating, google_rating, booking_rating,
		       recommended, admin_flag, quality, quiet, important_note,
		       type, chain_code, dupe_id, iata_code, last_update, created_at, updated_at
		FROM hotels
		ORDER BY hotel_id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hotels []*models.Hotel
	for rows.Next() {
		var hotel models.Hotel
		var latitude, longitude sql.NullFloat64
		var rec, adminFlag, quality, quiet int
		var amadeusRating, expediaRating, tripadvisorRating, googleRating, bookingRating sql.NullFloat64
		var sourceHotelID, name, city, country, streetAddress, postalCode, stateCode, phone, website, email, importantNote, hotelType, chainCode, iataCode, lastUpdate sql.NullString
		var dupeID sql.NullInt64

		err := rows.Scan(
			&hotel.ID,
			&hotel.HotelID,
			&hotel.Source,
			&sourceHotelID,
			&name,
			&city,
			&country,
			&latitude,
			&longitude,
			&streetAddress,
			&postalCode,
			&stateCode,
			&phone,
			&website,
			&email,
			&amadeusRating,
			&expediaRating,
			&tripadvisorRating,
			&googleRating,
			&bookingRating,
			&rec,
			&adminFlag,
			&quality,
			&quiet,
			&importantNote,
			&hotelType,
			&chainCode,
			&dupeID,
			&iataCode,
			&lastUpdate,
			&hotel.CreatedAt,
			&hotel.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if latitude.Valid {
			hotel.Latitude = &latitude.Float64
		}
		if longitude.Valid {
			hotel.Longitude = &longitude.Float64
		}
		if amadeusRating.Valid {
			hotel.AmadeusRating = &amadeusRating.Float64
		}
		if expediaRating.Valid {
			hotel.ExpediaRating = &expediaRating.Float64
		}
		if tripadvisorRating.Valid {
			hotel.TripadvisorRating = &tripadvisorRating.Float64
		}
		if googleRating.Valid {
			hotel.GoogleRating = &googleRating.Float64
		}
		if bookingRating.Valid {
			hotel.BookingRating = &bookingRating.Float64
		}

		hotel.Recommended = rec == 1
		hotel.AdminFlag = adminFlag == 1
		hotel.Quality = quality == 1
		hotel.Quiet = quiet == 1

		hotel.SourceHotelID = sourceHotelID.String
		hotel.Name = name.String
		hotel.City = city.String
		hotel.Country = country.String
		hotel.StreetAddress = streetAddress.String
		hotel.PostalCode = postalCode.String
		hotel.StateCode = stateCode.String
		hotel.Phone = phone.String
		hotel.Website = website.String
		hotel.Email = email.String
		hotel.ImportantNote = importantNote.String
		hotel.Type = hotelType.String
		hotel.ChainCode = chainCode.String
		if dupeID.Valid {
			hotel.DupeID = dupeID.Int64
		}
		hotel.IATACode = iataCode.String
		hotel.LastUpdate = lastUpdate.String

		hotels = append(hotels, &hotel)
	}

	return hotels, rows.Err()
}

// GetHotel retrieves a hotel by ID
func (s *Storage) GetHotel(ctx context.Context, hotelID string) (*models.Hotel, error) {
	query := `
		SELECT id, hotel_id, source, source_hotel_id, name, city, country,
		       latitude, longitude, street_address, postal_code, state_code, phone, website, email,
		       amadeus_rating, expedia_rating, tripadvisor_rating, google_rating, booking_rating,
		       recommended, admin_flag, quality, quiet, important_note,
		       type, chain_code, dupe_id, iata_code, last_update, created_at, updated_at
		FROM hotels
		WHERE hotel_id = ?
	`

	var hotel models.Hotel
	var rec, adminFlag, quality, quiet int
	var sourceHotelID, name, city, country, streetAddress, postalCode, stateCode, phone, website, email, importantNote, hotelType, chainCode, iataCode, lastUpdate sql.NullString
	var dupeID sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, hotelID).Scan(
		&hotel.ID,
		&hotel.HotelID,
		&hotel.Source,
		&sourceHotelID,
		&name,
		&city,
		&country,
		&hotel.Latitude,
		&hotel.Longitude,
		&streetAddress,
		&postalCode,
		&stateCode,
		&phone,
		&website,
		&email,
		&hotel.AmadeusRating,
		&hotel.ExpediaRating,
		&hotel.TripadvisorRating,
		&hotel.GoogleRating,
		&hotel.BookingRating,
		&rec,
		&adminFlag,
		&quality,
		&quiet,
		&importantNote,
		&hotelType,
		&chainCode,
		&dupeID,
		&iataCode,
		&lastUpdate,
		&hotel.CreatedAt,
		&hotel.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("hotel not found: %s", hotelID)
		}
		return nil, err
	}

	hotel.Recommended = rec == 1
	hotel.AdminFlag = adminFlag == 1
	hotel.Quality = quality == 1
	hotel.Quiet = quiet == 1

	hotel.SourceHotelID = sourceHotelID.String
	hotel.Name = name.String
	hotel.City = city.String
	hotel.Country = country.String
	hotel.StreetAddress = streetAddress.String
	hotel.PostalCode = postalCode.String
	hotel.StateCode = stateCode.String
	hotel.Phone = phone.String
	hotel.Website = website.String
	hotel.Email = email.String
	hotel.ImportantNote = importantNote.String
	hotel.Type = hotelType.String
	hotel.ChainCode = chainCode.String
	if dupeID.Valid {
		hotel.DupeID = dupeID.Int64
	}
	hotel.IATACode = iataCode.String
	hotel.LastUpdate = lastUpdate.String

	return &hotel, nil
}

// CreateOrUpdateHotel creates or updates a hotel in the consolidated hotels table
func (s *Storage) CreateOrUpdateHotel(ctx context.Context, hotel *models.Hotel) error {
	query := `
		INSERT INTO hotels (
			hotel_id, source, source_hotel_id, name, city, country,
			latitude, longitude, street_address, postal_code, state_code, phone, website, email,
			amadeus_rating, expedia_rating, tripadvisor_rating, google_rating, booking_rating,
			recommended, admin_flag, quality, quiet, important_note,
			type, chain_code, dupe_id, iata_code, address_json, geo_code_json, distance_json,
			last_update
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hotel_id) DO UPDATE SET
			source = excluded.source,
			source_hotel_id = excluded.source_hotel_id,
			name = excluded.name,
			city = excluded.city,
			country = excluded.country,
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			street_address = excluded.street_address,
			postal_code = excluded.postal_code,
		    state_code = excluded.state_code,
			phone = excluded.phone,
			website = excluded.website,
			email = excluded.email,
			amadeus_rating = COALESCE(excluded.amadeus_rating, hotels.amadeus_rating),
			expedia_rating = COALESCE(excluded.expedia_rating, hotels.expedia_rating),
			tripadvisor_rating = COALESCE(excluded.tripadvisor_rating, hotels.tripadvisor_rating),
			google_rating = COALESCE(excluded.google_rating, hotels.google_rating),
			booking_rating = COALESCE(excluded.booking_rating, hotels.booking_rating),
			important_note = excluded.important_note,
			type = excluded.type,
			chain_code = excluded.chain_code,
			dupe_id = excluded.dupe_id,
			iata_code = excluded.iata_code,
			address_json = excluded.address_json,
			geo_code_json = excluded.geo_code_json,
			distance_json = excluded.distance_json,
			last_update = excluded.last_update,
			updated_at = CURRENT_TIMESTAMP
	`

	recommended := 0
	if hotel.Recommended {
		recommended = 1
	}
	adminFlag := 0
	if hotel.AdminFlag {
		adminFlag = 1
	}
	quality := 0
	if hotel.Quality {
		quality = 1
	}
	quiet := 0
	if hotel.Quiet {
		quiet = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		hotel.HotelID,
		hotel.Source,
		hotel.SourceHotelID,
		hotel.Name,
		hotel.City,
		hotel.Country,
		hotel.Latitude,
		hotel.Longitude,
		hotel.StreetAddress,
		hotel.PostalCode,
		hotel.StateCode,
		hotel.Phone,
		hotel.Website,
		hotel.Email,
		hotel.AmadeusRating,
		hotel.ExpediaRating,
		hotel.TripadvisorRating,
		hotel.GoogleRating,
		hotel.BookingRating,
		recommended,
		adminFlag,
		quality,
		quiet,
		hotel.ImportantNote,
		hotel.Type,
		hotel.ChainCode,
		hotel.DupeID,
		hotel.IATACode,
		"", // address_json - can be populated from Amadeus data
		"", // geo_code_json
		"", // distance_json
		hotel.LastUpdate,
	)
	return err
}

// UpdateHotelRating updates rating from a specific source
func (s *Storage) UpdateHotelRating(ctx context.Context, hotelID, source string, rating float64) error {
	var column string
	switch source {
	case models.HotelSourceAmadeus:
		column = "amadeus_rating"
	case models.HotelSourceExpedia:
		column = "expedia_rating"
	case models.HotelSourceTripadvisor:
		column = "tripadvisor_rating"
	case models.HotelSourceGoogle:
		column = "google_rating"
	case models.HotelSourceBooking:
		column = "booking_rating"
	default:
		return fmt.Errorf("unknown source: %s", source)
	}

	query := fmt.Sprintf(`
		UPDATE hotels
		SET %s = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE hotel_id = ?
	`, column)

	_, err := s.db.ExecContext(ctx, query, rating, hotelID)
	return err
}

// UpdateRecommendationFields updates the recommendation fields for a hotel
func (s *Storage) UpdateRecommendationFields(ctx context.Context, hotelID string, recommended, quality, quiet bool, importantNote string) error {
	query := `
		UPDATE hotels
		SET recommended = ?,
		    quality = ?,
		    quiet = ?,
		    important_note = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE hotel_id = ?
	`

	rec := 0
	if recommended {
		rec = 1
	}
	q := 0
	if quality {
		q = 1
	}
	qui := 0
	if quiet {
		qui = 1
	}

	_, err := s.db.ExecContext(ctx, query, rec, q, qui, importantNote, hotelID)
	return err
}

// UpdateAdminFlag updates the admin flag for a hotel
func (s *Storage) UpdateAdminFlag(ctx context.Context, hotelID string, disabled bool) error {
	query := `
		UPDATE hotels
		SET admin_flag = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE hotel_id = ?
	`

	flag := 0
	if disabled {
		flag = 1
	}

	_, err := s.db.ExecContext(ctx, query, flag, hotelID)
	return err
}

// Create inserts a new hotel from Amadeus API
func (s *Storage) Create(ctx context.Context, hotel *models.HotelAPIItem) error {
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
		strings.Join(addressData.Line1, ", "), // Populated from parsed JSON
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

// GetHotelIDs returns all hotel IDs for processing
func (s *Storage) GetHotelIDs(ctx context.Context) ([]string, error) {
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
