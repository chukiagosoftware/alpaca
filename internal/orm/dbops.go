package orm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chukiagosoftware/alpaca/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetAllHotels retrieves all hotels from the database
func (db *DB) GetAllHotels(ctx context.Context, source string) ([]*models.Hotel, error) {
	var hotels []*models.Hotel
	if source != "" {
		return hotels, db.DB.WithContext(ctx).Where("source = ? and hotel_id > 'ta_633460'", source).Order("hotel_id").Find(&hotels).Error
	}
	return hotels, db.DB.WithContext(ctx).Order("hotel_id").Find(&hotels).Error
}

// GetHotel retrieves a hotel by ID
func (db *DB) GetHotel(ctx context.Context, hotelID string) (*models.Hotel, error) {
	var hotel models.Hotel
	err := db.DB.WithContext(ctx).Where("hotel_id = ?", hotelID).First(&hotel).Error
	return &hotel, err
}

// CreateOrUpdateHotel creates or updates a hotel in the consolidated hotels table
func (db *DB) CreateOrUpdateHotel(ctx context.Context, hotel *models.Hotel) error {
	var existing models.Hotel
	err := db.DB.WithContext(ctx).Where("hotel_id = ?", hotel.HotelID).First(&existing).Error
	if err == nil {
		// Record exists: update it
		return db.DB.WithContext(ctx).Model(&existing).Updates(hotel).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// Record doesn't exist: create it
		return db.DB.WithContext(ctx).Create(hotel).Error
	}
	return err // Other error (e.g., DB connection issue)
}

// UpdateHotelRating updates rating from a specific source
func (db *DB) UpdateHotelRating(ctx context.Context, hotelID, source string, rating float64) error {
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
	return db.DB.WithContext(ctx).Model(&models.Hotel{}).Where("hotel_id = ?", hotelID).Update(column, rating).Error
}

// UpdateRecommendationFields updates the recommendation fields for a hotel
func (db *DB) UpdateRecommendationFields(ctx context.Context, hotelID string, recommended, quality, quiet bool, importantNote string) error {
	updates := map[string]interface{}{
		"recommended":    recommended,
		"quality":        quality,
		"quiet":          quiet,
		"important_note": importantNote,
	}
	return db.DB.WithContext(ctx).Model(&models.Hotel{}).Where("hotel_id = ?", hotelID).Updates(updates).Error
}

// UpdateAdminFlag updates the admin flag for a hotel
func (db *DB) UpdateAdminFlag(ctx context.Context, hotelID string, disabled bool) error {
	return db.DB.WithContext(ctx).Model(&models.Hotel{}).Where("hotel_id = ?", hotelID).Update("admin_flag", disabled).Error
}

// Create inserts a new hotel from Amadeus API
func (db *DB) Create(ctx context.Context, apiHotel *models.HotelAPIItem) error {
	var addressData struct {
		CityName    string   `json:"cityName"`
		CountryCode string   `json:"countryCode"`
		Lines       []string `json:"lines"`
		PostalCode  string   `json:"postalCode"`
		StateCode   string   `json:"stateCode"`
	}
	json.Unmarshal(apiHotel.Address, &addressData)

	hotel := &models.Hotel{
		HotelID:       apiHotel.HotelID,
		Source:        models.HotelSourceAmadeus,
		SourceHotelID: apiHotel.HotelID,
		Name:          apiHotel.Name,
		City:          addressData.CityName,
		Country:       addressData.CountryCode,
		StreetAddress: strings.Join(addressData.Lines, ", "),
		PostalCode:    addressData.PostalCode,
		StateCode:     addressData.StateCode,
		Type:          apiHotel.Type,
		DupeID:        int64(apiHotel.DupeID),
		IATACode:      apiHotel.IATACode,
		LastUpdate:    apiHotel.LastUpdate,
	}

	return db.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "hotel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"source", "name", "city", "country", "street_address",
			"postal_code", "state_code", "type", "chain_code",
			"dupe_id", "iata_code", "last_update",
		}),
	}).Create(hotel).Error
}

// GetHotelIDs returns all hotel IDs for processing
func (db *DB) GetHotelIDs(ctx context.Context) ([]string, error) {
	var hotelIDs []string
	return hotelIDs, db.DB.WithContext(ctx).Model(&models.Hotel{}).Pluck("hotel_id", &hotelIDs).Error
}

// UpdateAmadeusSearchData updates a hotel with Amadeus search data
func (db *DB) UpdateAmadeusSearchData(ctx context.Context, hotelID string, data *models.HotelSearchData) error {
	return db.DB.WithContext(ctx).Model(&models.Hotel{}).Where("hotel_id = ?", hotelID).Updates(map[string]interface{}{
		"amadeus_rating": data.Rating,
		"last_update":    data.LastUpdate,
	}).Error
}

// UpdateAmadeusRatingsData updates a hotel with Amadeus ratings data
func (db *DB) UpdateAmadeusRatingsData(ctx context.Context, hotelID string, data *models.HotelRatingsData) error {
	sentimentsJSON, _ := json.Marshal(data.Sentiments)
	return db.DB.WithContext(ctx).Model(&models.Hotel{}).Where("hotel_id = ?", hotelID).Updates(map[string]interface{}{
		"amadeus_number_of_reviews": data.NumberOfReviews,
		"amadeus_number_of_ratings": data.NumberOfRatings,
		"amadeus_overall_rating":    data.OverallRating,
		"amadeus_sentiments":        string(sentimentsJSON),
		"last_ratings_update":       data.LastUpdate,
	}).Error
}

// SaveReview saves a review to the database

// SaveReview saves a review to the database

func (db *DB) SaveReview(ctx context.Context, review *models.HotelReview) error {
	return db.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "source_review_id"}},
		DoNothing: true,
	}).Create(review).Error
}

// GetReviewsForHotel retrieves all reviews for a hotel
func (db *DB) GetReviewsForHotel(ctx context.Context, hotelID string) ([]*models.HotelReview, error) {
	var reviews []*models.HotelReview
	err := db.DB.WithContext(ctx).Where("hotel_id = ?", hotelID).Order("review_date DESC, created_at DESC").Find(&reviews).Error
	return reviews, err
}

// GetReviewTexts returns just the review texts for LLM processing
func (db *DB) GetReviewTexts(ctx context.Context, hotelID string) ([]string, error) {
	reviews, err := db.GetReviewsForHotel(ctx, hotelID)
	if err != nil {
		return nil, err
	}

	texts := make([]string, 0, len(reviews))
	for _, review := range reviews {
		if strings.TrimSpace(review.ReviewText) != "" {
			texts = append(texts, review.ReviewText)
		}
	}

	return texts, nil
}
