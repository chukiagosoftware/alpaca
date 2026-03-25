package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	//"gorm.io/gorm/clause"
)

// ReviewSource Need to go back and fix for google
// ReviewSource defines the interface for review sources
type ReviewSource interface {
	GetSourceName() string
	FetchReviewsForLocation(context.Context, string, *gorm.DB) ([]*models.HotelReview, error)
}

// ReviewCrawlerService handles crawling reviews from multiple sources
type ReviewCrawlerService struct {
	db *gorm.DB
}

// NewReviewCrawlerService creates a new review crawler service
func NewReviewCrawlerService(db *gorm.DB) *ReviewCrawlerService {
	return &ReviewCrawlerService{db: db}
}

// CrawlAllSources crawls reviews from all available sources for a hotel
func (s *ReviewCrawlerService) CrawlReviews(ctx context.Context, hotel *models.Hotel) (int, error) {
	services := map[string]ReviewSource{
		models.SourceTripadvisor: NewTripAdvisorReviewsService(),
		//models.SourceGoogle:      NewGoogleCrawler(),
		//NewExpediaCrawler(),
		//NewBookingCrawler(),
		//NewHotelWebsiteCrawler(),
		//NewBingCrawler(),
		//NewYelpCrawler(),
	}

	totalReviews := 0
	service := services[hotel.Source]
	reviews, err := service.FetchReviewsForLocation(ctx, hotel.SourceHotelID, s.db)
	if err != nil {
		log.Printf("Error crawling %s reviews for hotel %s: %v", service.GetSourceName(), hotel.SourceHotelID, err)
	}

	for _, review := range reviews {
		if err := s.SaveReview(ctx, review); err != nil {
			log.Printf("Error saving review from %s: %v", service.GetSourceName(), err)
			continue
		}
		totalReviews++
	}

	// Rate limiting between sources
	time.Sleep(1 * time.Second)

	return totalReviews, nil
}

// SaveReview saves a review to the database
func (s *ReviewCrawlerService) SaveReview(ctx context.Context, review *models.HotelReview) error {
	err := s.db.WithContext(ctx).
		Where("source_review_id = ? AND source = ?", review.SourceReviewID, review.Source).
		FirstOrCreate(&review).Error

	if err != nil {
		return fmt.Errorf("failed to save review %s: %w", review.SourceReviewID, err)
	}

	return nil
}

// GetReviewsForHotel retrieves all reviews for a hotel
func (s *ReviewCrawlerService) GetReviewsForHotel(ctx context.Context, hotelID string) ([]*models.HotelReview, error) {
	var reviews []*models.HotelReview
	err := s.db.WithContext(ctx).Where("hotel_id = ?", hotelID).Order("review_date DESC, created_at DESC").Find(&reviews).Error
	return reviews, err
}

// GetReviewTexts returns just the review texts for LLM processing
func (s *ReviewCrawlerService) GetReviewTexts(ctx context.Context, hotelID string) ([]string, error) {
	reviews, err := s.GetReviewsForHotel(ctx, hotelID)
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

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(currentFile))
	envPath := filepath.Join(projectRoot, ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	crawler := NewReviewCrawlerService(db.DB)

	ctx := context.Background()

	// Get all hotels
	hotels, err := db.GetAllHotels(ctx, models.SourceTripadvisor) // "" for all hotels
	if err != nil {
		log.Fatalf("Failed to get hotels: %v", err)
	}
	log.Printf("Found %d hotels to fetch reviews for.", len(hotels))

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalReviewsCrawled := 0
	const maxGo = 5
	sem := make(chan struct{}, maxGo)

	for _, hotel := range hotels {
		wg.Add(1)
		sem <- struct{}{}
		go func(h *models.Hotel) {
			defer wg.Done()
			defer func() { <-sem }()

			log.Printf("Fetching reviews for %s (%s, %s)", h.Name, h.HotelID, h.Source)
			reviewsCount, err := crawler.CrawlReviews(ctx, h)
			if err != nil {
				log.Printf("Error fetching %s reviews for hotel %s(%s): %v", h.Source, h.Name, h.SourceHotelID, err)
				return
			}
			mu.Lock()
			totalReviewsCrawled += reviewsCount
			mu.Unlock()
			log.Printf("Fetched %d reviews for hotel %s", reviewsCount, h.SourceHotelID)

		}(hotel)

		wg.Wait()

		log.Printf("Fetched %d reviews so far for %s", totalReviewsCrawled, hotel.SourceHotelID)
		// Rate limiting between hotels
		time.Sleep(800 * time.Millisecond)
	}

	log.Printf("Review fetch completed. Total New Reviews: %d", totalReviewsCrawled)
}
