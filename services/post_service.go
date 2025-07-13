package services

import (
	"context"

	"github.com/edamsoft-sre/alpaca/models"
	"gorm.io/gorm"
)

type PostService struct {
	db *gorm.DB
}

func NewPostService(db *gorm.DB) *PostService {
	return &PostService{db: db}
}

func (s *PostService) Create(ctx context.Context, post *models.Post) error {
	return s.db.WithContext(ctx).Create(post).Error
}

func (s *PostService) GetByID(ctx context.Context, id string) (*models.Post, error) {
	var post models.Post
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&post).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *PostService) Delete(ctx context.Context, id string, userId string) error {
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userId).Delete(&models.Post{}).Error
}

func (s *PostService) Update(ctx context.Context, post *models.Post, userId string) error {
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", post.Id, userId).Updates(post).Error
}

func (s *PostService) List(ctx context.Context, page uint64) ([]*models.Post, error) {
	var posts []*models.Post
	limit := 5
	offset := int(page) * limit

	err := s.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}
