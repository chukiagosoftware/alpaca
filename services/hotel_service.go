package services

import (
	"context"
	"encoding/json"

	"github.com/edamsoft-sre/alpaca/database"
	"github.com/edamsoft-sre/alpaca/models"
)

type HotelService struct {
	db *database.DB
}

func NewHotelService(db *database.DB) *HotelService {
	return &HotelService{db: db}
}

// Create inserts a new hotel
func (s *HotelService) Create(ctx context.Context, hotel *models.HotelAPIItem) error {
	query := `
		INSERT INTO hotels (hotel_id, type, chain_code, dupe_id, name, iata_code, address, geo_code, distance, last_update)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hotel_id) DO UPDATE SET
			type = excluded.type,
			chain_code = excluded.chain_code,
			dupe_id = excluded.dupe_id,
			name = excluded.name,
			iata_code = excluded.iata_code,
			address = excluded.address,
			geo_code = excluded.geo_code,
			distance = excluded.distance,
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
		string(addressJSON),
		string(geoCodeJSON),
		string(distanceJSON),
		hotel.LastUpdate,
	)
	return err
}

// GetHotelIDs returns all hotel IDs for processing
func (s *HotelService) GetHotelIDs(ctx context.Context) ([]string, error) {
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

// UpsertSearchData inserts or updates hotel search data
func (s *HotelService) UpsertSearchData(ctx context.Context, searchData *models.HotelSearchData) error {
	query := `
		INSERT INTO hotel_search_data (
			hotel_id, type, chain_code, dupe_id, name, rating, official_rating,
			description, media, amenities, address, contact, policies, available,
			offers, self, hotel_distance, last_update
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hotel_id) DO UPDATE SET
			type = excluded.type,
			chain_code = excluded.chain_code,
			dupe_id = excluded.dupe_id,
			name = excluded.name,
			rating = excluded.rating,
			official_rating = excluded.official_rating,
			description = excluded.description,
			media = excluded.media,
			amenities = excluded.amenities,
			address = excluded.address,
			contact = excluded.contact,
			policies = excluded.policies,
			available = excluded.available,
			offers = excluded.offers,
			self = excluded.self,
			hotel_distance = excluded.hotel_distance,
			last_update = excluded.last_update,
			updated_at = CURRENT_TIMESTAMP
	`

	descriptionJSON, _ := json.Marshal(searchData.Description)
	mediaJSON, _ := json.Marshal(searchData.Media)
	amenitiesJSON, _ := json.Marshal(searchData.Amenities)
	addressJSON, _ := json.Marshal(searchData.Address)
	contactJSON, _ := json.Marshal(searchData.Contact)
	policiesJSON, _ := json.Marshal(searchData.Policies)
	offersJSON, _ := json.Marshal(searchData.Offers)
	hotelDistanceJSON, _ := json.Marshal(searchData.HotelDistance)

	available := 0
	if searchData.Available {
		available = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		searchData.HotelID,
		searchData.Type,
		searchData.ChainCode,
		searchData.DupeID,
		searchData.Name,
		searchData.Rating,
		searchData.OfficialRating,
		string(descriptionJSON),
		string(mediaJSON),
		string(amenitiesJSON),
		string(addressJSON),
		string(contactJSON),
		string(policiesJSON),
		available,
		string(offersJSON),
		searchData.Self,
		string(hotelDistanceJSON),
		searchData.LastUpdate,
	)
	return err
}

// UpsertRatingsData inserts or updates hotel ratings data
func (s *HotelService) UpsertRatingsData(ctx context.Context, ratingsData *models.HotelRatingsData) error {
	query := `
		INSERT INTO hotel_ratings_data (
			hotel_id, type, number_of_reviews, number_of_ratings,
			overall_rating, sentiments, last_update
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hotel_id) DO UPDATE SET
			type = excluded.type,
			number_of_reviews = excluded.number_of_reviews,
			number_of_ratings = excluded.number_of_ratings,
			overall_rating = excluded.overall_rating,
			sentiments = excluded.sentiments,
			last_update = excluded.last_update,
			updated_at = CURRENT_TIMESTAMP
	`

	sentimentsJSON, _ := json.Marshal(ratingsData.Sentiments)

	_, err := s.db.ExecContext(ctx, query,
		ratingsData.HotelID,
		ratingsData.Type,
		ratingsData.NumberOfReviews,
		ratingsData.NumberOfRatings,
		ratingsData.OverallRating,
		string(sentimentsJSON),
		ratingsData.LastUpdate,
	)
	return err
}

// IsHotelIDInvalidForSearch checks if a hotel ID is marked as invalid
func (s *HotelService) IsHotelIDInvalidForSearch(ctx context.Context, hotelID string) (bool, error) {
	query := `SELECT COUNT(*) FROM invalid_hotel_search_ids WHERE hotel_id = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, hotelID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkHotelIDInvalidForSearch marks a hotel ID as invalid for search
func (s *HotelService) MarkHotelIDInvalidForSearch(ctx context.Context, hotelID string) error {
	query := `
		INSERT INTO invalid_hotel_search_ids (hotel_id)
		VALUES (?)
		ON CONFLICT(hotel_id) DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, query, hotelID)
	return err
}
