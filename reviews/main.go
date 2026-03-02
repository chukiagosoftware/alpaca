package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReviewSource defines the interface for review sources
type ReviewSource interface {
	GetSourceName() string
	CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error)
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
func (s *ReviewCrawlerService) CrawlAllSources(ctx context.Context, hotel *models.Hotel) (int, error) {
	sources := []ReviewSource{
		//NewTripadvisorCrawler(),
		NewGoogleCrawler(), //
		//NewExpediaCrawler(),
		//NewBookingCrawler(),
		//NewHotelWebsiteCrawler(),
		//NewBingCrawler(),
		//NewYelpCrawler(),
	}

	totalReviews := 0
	for _, source := range sources {
		reviews, err := source.CrawlReviews(ctx, hotel)
		if err != nil {
			log.Printf("Error crawling %s reviews for hotel %s: %v", source.GetSourceName(), hotel.HotelID, err)
			continue
		}

		for _, review := range reviews {
			if err := s.SaveReview(ctx, review); err != nil {
				log.Printf("Error saving review from %s: %v", source.GetSourceName(), err)
				continue
			}
			totalReviews++
		}

		// Rate limiting between sources
		time.Sleep(1 * time.Second)
	}

	return totalReviews, nil
}

// SaveReview saves a review to the database
func (s *ReviewCrawlerService) SaveReview(ctx context.Context, review *models.HotelReview) error {
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "hotel_id"}, {Name: "source"}, {Name: "source_review_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"reviewer_name", "reviewer_location", "rating", "review_text",
			"review_date", "verified", "helpful_count", "room_type", "travel_type", "stay_date",
		}),
	}).Create(review).Error
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

	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	crawler := NewReviewCrawlerService(db.DB)

	ctx := context.Background()

	// Get all hotels
	hotelDB := database.NewStorage(db.DB)
	hotels, err := hotelDB.GetAllHotels(ctx)
	if err != nil {
		log.Fatalf("Failed to get hotels: %v", err)
	}

	// Filter to only Google-sourced hotels
	var googleHotels []*models.Hotel
	for _, hotel := range hotels {
		if hotel.Source == models.HotelSourceGoogle {
			googleHotels = append(googleHotels, hotel)
		}
	}
	hotels = googleHotels

	log.Printf("Found %d Google-sourced hotels to crawl reviews for", len(hotels))

	totalReviewsCrawled := 0
	for i, hotel := range hotels {
		log.Printf("Crawling reviews for hotel %d/%d: %s (%s)", i+1, len(hotels), hotel.Name, hotel.HotelID)

		reviewsCount, err := crawler.CrawlAllSources(ctx, hotel)
		if err != nil {
			log.Printf("Error crawling reviews for hotel %s: %v", hotel.HotelID, err)
			continue
		}

		totalReviewsCrawled += reviewsCount
		log.Printf("Crawled %d reviews for hotel %s", reviewsCount, hotel.HotelID)

		// Rate limiting between hotels
		time.Sleep(2 * time.Second)
	}

	log.Printf("Review crawling completed. Total reviews crawled: %d", totalReviewsCrawled)
}
