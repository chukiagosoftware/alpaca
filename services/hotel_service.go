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

func (s *HotelService) List(ctx context.Context, page uint64, limit uint64) ([]*models.HotelAPIItem, error) {
	var hotels []*models.HotelAPIItem
	offset := int(page) * int(limit)

	err := s.db.WithContext(ctx).Limit(int(limit)).Offset(offset).Find(&hotels).Error
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
	err := s.db.WithContext(ctx).Where("hotel_id = ?", hotelID).First(&searchData).Error
	if err != nil {
		return nil, err
	}
	return &searchData, nil
}

func (s *HotelService) UpdateSearchData(ctx context.Context, searchData *models.HotelSearchData) error {
	return s.db.WithContext(ctx).Save(searchData).Error
}

// HotelRatingsData methods
func (s *HotelService) CreateRatingsData(ctx context.Context, ratingsData *models.HotelRatingsData) error {
	return s.db.WithContext(ctx).Create(ratingsData).Error
}

func (s *HotelService) GetRatingsDataByHotelID(ctx context.Context, hotelID string) (*models.HotelRatingsData, error) {
	var ratingsData models.HotelRatingsData
	err := s.db.WithContext(ctx).Where("hotel_id = ?", hotelID).First(&ratingsData).Error
	if err != nil {
		return nil, err
	}
	return &ratingsData, nil
}

func (s *HotelService) UpdateRatingsData(ctx context.Context, ratingsData *models.HotelRatingsData) error {
	return s.db.WithContext(ctx).Save(ratingsData).Error
}

// GetHotelWithDetails returns hotel with search and ratings data
func (s *HotelService) GetHotelWithDetails(ctx context.Context, hotelID string) (*models.HotelAPIItem, *models.HotelSearchData, *models.HotelRatingsData, error) {
	hotel, err := s.GetByID(ctx, hotelID)
	if err != nil {
		return nil, nil, nil, err
	}

	searchData, err := s.GetSearchDataByHotelID(ctx, hotelID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, nil, nil, err
	}

	ratingsData, err := s.GetRatingsDataByHotelID(ctx, hotelID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, nil, nil, err
	}

	return hotel, searchData, ratingsData, nil
}
