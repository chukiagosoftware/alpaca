package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/edamsoft-sre/alpaca/alpaca/models"
)

// CreateOrUpdateHotel creates or updates a hotel in the consolidated hotels table
func (s *HotelService) CreateOrUpdateHotel(ctx context.Context, hotel *models.Hotel) error {
	query := `
		INSERT INTO hotels (
			hotel_id, source, source_hotel_id, name, city, country,
			latitude, longitude, street_address, postal_code, phone, website, email,
			amadeus_rating, expedia_rating, tripadvisor_rating, google_rating, booking_rating,
			recommended, admin_flag, quality, quiet, important_note,
			type, chain_code, dupe_id, iata_code, address_json, geo_code_json, distance_json,
			last_update
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

// UpdateRecommendationFields updates the recommendation fields for a hotel
func (s *HotelService) UpdateRecommendationFields(ctx context.Context, hotelID string, recommended, quality, quiet bool, importantNote string) error {
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
func (s *HotelService) UpdateAdminFlag(ctx context.Context, hotelID string, disabled bool) error {
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

// GetHotel retrieves a hotel by ID
func (s *HotelService) GetHotel(ctx context.Context, hotelID string) (*models.Hotel, error) {
	query := `
		SELECT id, hotel_id, source, source_hotel_id, name, city, country,
		       latitude, longitude, street_address, postal_code, phone, website, email,
		       amadeus_rating, expedia_rating, tripadvisor_rating, google_rating, booking_rating,
		       recommended, admin_flag, quality, quiet, important_note,
		       type, chain_code, dupe_id, iata_code, last_update, created_at, updated_at
		FROM hotels
		WHERE hotel_id = ?
	`

	var hotel models.Hotel
	var rec, adminFlag, quality, quiet int

	err := s.db.QueryRowContext(ctx, query, hotelID).Scan(
		&hotel.ID,
		&hotel.HotelID,
		&hotel.Source,
		&hotel.SourceHotelID,
		&hotel.Name,
		&hotel.City,
		&hotel.Country,
		&hotel.Latitude,
		&hotel.Longitude,
		&hotel.StreetAddress,
		&hotel.PostalCode,
		&hotel.Phone,
		&hotel.Website,
		&hotel.Email,
		&hotel.AmadeusRating,
		&hotel.ExpediaRating,
		&hotel.TripadvisorRating,
		&hotel.GoogleRating,
		&hotel.BookingRating,
		&rec,
		&adminFlag,
		&quality,
		&quiet,
		&hotel.ImportantNote,
		&hotel.Type,
		&hotel.ChainCode,
		&hotel.DupeID,
		&hotel.IATACode,
		&hotel.LastUpdate,
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

	return &hotel, nil
}

// SaveRecommendation saves LLM-processed recommendation
func (s *HotelService) SaveRecommendation(ctx context.Context, rec *models.HotelRecommendation) error {
	query := `
		INSERT INTO hotel_recommendations (
			hotel_id, quality_score, quality_confidence, quality_reasoning,
			quiet_score, quiet_confidence, quiet_reasoning,
			overall_recommended, recommendation_summary, reviews_analyzed, llm_model
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hotel_id) DO UPDATE SET
			quality_score = excluded.quality_score,
			quality_confidence = excluded.quality_confidence,
			quality_reasoning = excluded.quality_reasoning,
			quiet_score = excluded.quiet_score,
			quiet_confidence = excluded.quiet_confidence,
			quiet_reasoning = excluded.quiet_reasoning,
			overall_recommended = excluded.overall_recommended,
			recommendation_summary = excluded.recommendation_summary,
			reviews_analyzed = excluded.reviews_analyzed,
			llm_model = excluded.llm_model,
			updated_at = CURRENT_TIMESTAMP
	`

	overallRec := 0
	if rec.OverallRecommended {
		overallRec = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		rec.HotelID,
		rec.QualityScore,
		rec.QualityConfidence,
		rec.QualityReasoning,
		rec.QuietScore,
		rec.QuietConfidence,
		rec.QuietReasoning,
		overallRec,
		rec.RecommendationSummary,
		rec.ReviewsAnalyzed,
		rec.LLMModel,
	)
	return err
}

// GetRecommendation retrieves recommendation for a hotel
func (s *HotelService) GetRecommendation(ctx context.Context, hotelID string) (*models.HotelRecommendation, error) {
	query := `
		SELECT id, hotel_id, quality_score, quality_confidence, quality_reasoning,
		       quiet_score, quiet_confidence, quiet_reasoning,
		       overall_recommended, recommendation_summary, reviews_analyzed, llm_model,
		       processed_at, created_at, updated_at
		FROM hotel_recommendations
		WHERE hotel_id = ?
	`

	var rec models.HotelRecommendation
	var overallRec int

	err := s.db.QueryRowContext(ctx, query, hotelID).Scan(
		&rec.ID,
		&rec.HotelID,
		&rec.QualityScore,
		&rec.QualityConfidence,
		&rec.QualityReasoning,
		&rec.QuietScore,
		&rec.QuietConfidence,
		&rec.QuietReasoning,
		&overallRec,
		&rec.RecommendationSummary,
		&rec.ReviewsAnalyzed,
		&rec.LLMModel,
		&rec.ProcessedAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No recommendation yet
		}
		return nil, err
	}

	rec.OverallRecommended = overallRec == 1
	return &rec, nil
}

// UpdateHotelRating updates rating from a specific source
func (s *HotelService) UpdateHotelRating(ctx context.Context, hotelID, source string, rating float64) error {
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
