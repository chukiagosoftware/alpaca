package services

import (
	"context"

	"github.com/edamsoft-sre/alpaca/models"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) Create(ctx context.Context, user *models.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *UserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
