package services

import (
	"context"

	"github.com/edamsoft-sre/alpaca/models"
	"gorm.io/gorm"
)

type HotelService struct {
	db *gorm.DB
}

func NewHotelService(db *gorm.DB) *HotelService {
	return &HotelService{db: db}
}

// HotelAPIItem methods
func (s *HotelService) Create(ctx context.Context, hotel *models.HotelAPIItem) error {
	return s.db.WithContext(ctx).Create(hotel).Error
}

func (s *HotelService) GetByID(ctx context.Context, id string) (*models.HotelAPIItem, error) {
	var hotel models.HotelAPIItem
	err := s.db.WithContext(ctx).Where("hotel_id = ?", id).First(&hotel).Error
	if err != nil {
		return nil, err
	}
	return &hotel, nil
}

func (s *HotelService) GetByIDWithDetails(ctx context.Context, id string) (*models.HotelAPIItem, error) {
	var hotel models.HotelAPIItem
	err := s.db.WithContext(ctx).
		Preload("SearchData").
		Preload("RatingsData").
		Where("hotel_id = ?", id).
		First(&hotel).Error
	if err != nil {
		return nil, err
	}
	return &hotel, nil
}

func (s *HotelService) List(ctx context.Context, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	err := s.db.WithContext(ctx).Limit(int(limit)).Offset(offset).Find(&hotels).Error
	if err != nil {
		return nil, err
	}
	return hotels, nil
}

func (s *HotelService) ListWithDetails(ctx context.Context, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	err := s.db.WithContext(ctx).
		Preload("SearchData").
		Preload("RatingsData").
		Limit(int(limit)).
		Offset(offset).
		Find(&hotels).Error
	if err != nil {
		return nil, err
	}
	return hotels, nil
}

func (s *HotelService) GetByCity(ctx context.Context, cityName string, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	// For PostgreSQL, use JSON operators
	if s.db.Dialector.Name() == "postgres" {
		err := s.db.WithContext(ctx).Where("address->>'cityName' ILIKE ?", "%"+cityName+"%").Limit(int(limit)).Offset(offset).Find(&hotels).Error
		if err != nil {
			return nil, err
		}
	} else {
		// For SQLite, use LIKE on the JSON string
		err := s.db.WithContext(ctx).Where("address LIKE ?", "%"+cityName+"%").Limit(int(limit)).Offset(offset).Find(&hotels).Error
		if err != nil {
			return nil, err
		}
	}
	return hotels, nil
}

func (s *HotelService) GetByCityWithDetails(ctx context.Context, cityName string, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	// For PostgreSQL, use JSON operators
	if s.db.Dialector.Name() == "postgres" {
		err := s.db.WithContext(ctx).
			Preload("SearchData").
			Preload("RatingsData").
			Where("address->>'cityName' ILIKE ?", "%"+cityName+"%").
			Limit(int(limit)).
			Offset(offset).
			Find(&hotels).Error
		if err != nil {
			return nil, err
		}
	} else {
		// For SQLite, use LIKE on the JSON string
		err := s.db.WithContext(ctx).
			Preload("SearchData").
			Preload("RatingsData").
			Where("address LIKE ?", "%"+cityName+"%").
			Limit(int(limit)).
			Offset(offset).
			Find(&hotels).Error
		if err != nil {
			return nil, err
		}
	}
	return hotels, nil
}

func (s *HotelService) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.HotelAPIItem{}).Count(&count).Error
	return count, err
}

// GetHotelIDs returns all hotel IDs for processing
func (s *HotelService) GetHotelIDs(ctx context.Context) ([]string, error) {
	var hotelIDs []string
	err := s.db.WithContext(ctx).Model(&models.HotelAPIItem{}).Pluck("hotel_id", &hotelIDs).Error
	return hotelIDs, err
}

// HotelSearchData methods
func (s *HotelService) CreateSearchData(ctx context.Context, searchData *models.HotelSearchData) error {
	return s.db.WithContext(ctx).Create(searchData).Error
}

func (s *HotelService) GetSearchDataByHotelID(ctx context.Context, hotelID string) (*models.HotelSearchData, error) {
	var searchData models.HotelSearchData
	err := s.db.WithContext(ctx).
		Preload("Hotel").
		Where("hotel_id = ?", hotelID).
		First(&searchData).Error
	if err != nil {
		return nil, err
	}
	return &searchData, nil
}

func (s *HotelService) UpdateSearchData(ctx context.Context, searchData *models.HotelSearchData) error {
	return s.db.WithContext(ctx).Save(searchData).Error
}

func (s *HotelService) UpsertSearchData(ctx context.Context, searchData *models.HotelSearchData) error {
	// Try to update first, if not found then create
	result := s.db.WithContext(ctx).Where("hotel_id = ?", searchData.HotelID).Updates(searchData)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// No rows were updated, so create new record
		return s.db.WithContext(ctx).Create(searchData).Error
	}
	return nil
}

// HotelRatingsData methods
func (s *HotelService) CreateRatingsData(ctx context.Context, ratingsData *models.HotelRatingsData) error {
	return s.db.WithContext(ctx).Create(ratingsData).Error
}

func (s *HotelService) GetRatingsDataByHotelID(ctx context.Context, hotelID string) (*models.HotelRatingsData, error) {
	var ratingsData models.HotelRatingsData
	err := s.db.WithContext(ctx).
		Preload("Hotel").
		Where("hotel_id = ?", hotelID).
		First(&ratingsData).Error
	if err != nil {
		return nil, err
	}
	return &ratingsData, nil
}

func (s *HotelService) UpdateRatingsData(ctx context.Context, ratingsData *models.HotelRatingsData) error {
	return s.db.WithContext(ctx).Save(ratingsData).Error
}

func (s *HotelService) UpsertRatingsData(ctx context.Context, ratingsData *models.HotelRatingsData) error {
	// Try to update first, if not found then create
	result := s.db.WithContext(ctx).Where("hotel_id = ?", ratingsData.HotelID).Updates(ratingsData)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// No rows were updated, so create new record
		return s.db.WithContext(ctx).Create(ratingsData).Error
	}
	return nil
}

// Check if a hotel ID is marked as invalid for the Search API
func (s *HotelService) IsHotelIDInvalidForSearch(ctx context.Context, hotelID string) (bool, error) {
	var invalid models.InvalidHotelSearchID
	err := s.db.WithContext(ctx).Where("hotel_id = ?", hotelID).First(&invalid).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Mark a hotel ID as invalid for the Search API
func (s *HotelService) MarkHotelIDInvalidForSearch(ctx context.Context, hotelID string) error {
	invalid := models.InvalidHotelSearchID{HotelID: hotelID}
	return s.db.WithContext(ctx).FirstOrCreate(&invalid, models.InvalidHotelSearchID{HotelID: hotelID}).Error
}

// GetHotelWithDetails returns hotel with search and ratings data
// This method is kept for backward compatibility but GetByIDWithDetails is preferred
func (s *HotelService) GetHotelWithDetails(ctx context.Context, hotelID string) (*models.HotelAPIItem, *models.HotelSearchData, *models.HotelRatingsData, error) {
	hotel, err := s.GetByIDWithDetails(ctx, hotelID)
	if err != nil {
		return nil, nil, nil, err
	}

	return hotel, hotel.SearchData, hotel.RatingsData, nil
}

// GetHotelsWithCompleteData returns hotels that have all three types of data
func (s *HotelService) GetHotelsWithCompleteData(ctx context.Context, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	err := s.db.WithContext(ctx).
		Preload("SearchData").
		Preload("RatingsData").
		Joins("JOIN hotel_search_data ON hotel_api_items.hotel_id = hotel_search_data.hotel_id").
		Joins("JOIN hotel_ratings_data ON hotel_api_items.hotel_id = hotel_ratings_data.hotel_id").
		Limit(int(limit)).
		Offset(offset).
		Find(&hotels).Error
	if err != nil {
		return nil, err
	}
	return hotels, nil
}

// GetHotelsWithSearchData returns hotels that have search data
func (s *HotelService) GetHotelsWithSearchData(ctx context.Context, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	err := s.db.WithContext(ctx).
		Preload("SearchData").
		Joins("JOIN hotel_search_data ON hotel_api_items.hotel_id = hotel_search_data.hotel_id").
		Limit(int(limit)).
		Offset(offset).
		Find(&hotels).Error
	if err != nil {
		return nil, err
	}
	return hotels, nil
}

// GetHotelsWithRatingsData returns hotels that have ratings data
func (s *HotelService) GetHotelsWithRatingsData(ctx context.Context, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	err := s.db.WithContext(ctx).
		Preload("RatingsData").
		Joins("JOIN hotel_ratings_data ON hotel_api_items.hotel_id = hotel_ratings_data.hotel_id").
		Limit(int(limit)).
		Offset(offset).
		Find(&hotels).Error
	if err != nil {
		return nil, err
	}
	return hotels, nil
}
