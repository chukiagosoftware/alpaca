package main

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/database"
	"github.com/chukiagosoftware/alpaca/internal/hotelstorage"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
)

// ReviewSource defines the interface for review sources
type ReviewSource interface {
	GetSourceName() string
	CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error)
}

// ReviewCrawlerService handles crawling reviews from multiple sources
type ReviewCrawlerService struct {
	db *database.DB
}

// NewReviewCrawlerService creates a new review crawler service
func NewReviewCrawlerService(db *database.DB) *ReviewCrawlerService {
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
	query := `
		INSERT INTO hotel_reviews (
			hotel_id, source, source_review_id, reviewer_name, reviewer_location,
			rating, review_text, review_date, verified, helpful_count,
			room_type, travel_type, stay_date
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hotel_id, source, source_review_id) DO UPDATE SET
			reviewer_name = excluded.reviewer_name,
			reviewer_location = excluded.reviewer_location,
			rating = excluded.rating,
			review_text = excluded.review_text,
			review_date = excluded.review_date,
			verified = excluded.verified,
			helpful_count = excluded.helpful_count,
			room_type = excluded.room_type,
			travel_type = excluded.travel_type,
			stay_date = excluded.stay_date,
			updated_at = CURRENT_TIMESTAMP
	`

	var reviewDate, stayDate interface{}
	if review.ReviewDate != nil {
		reviewDate = review.ReviewDate.Format(time.RFC3339)
	}
	if review.StayDate != nil {
		stayDate = review.StayDate.Format(time.RFC3339)
	}

	verified := 0
	if review.Verified {
		verified = 1
	}

	var rating interface{}
	if review.Rating != nil {
		rating = *review.Rating
	}

	_, err := s.db.ExecContext(ctx, query,
		review.HotelID,
		review.Source,
		review.SourceReviewID,
		review.ReviewerName,
		review.ReviewerLocation,
		rating,
		review.ReviewText,
		reviewDate,
		verified,
		review.HelpfulCount,
		review.RoomType,
		review.TravelType,
		stayDate,
	)
	return err
}

// GetReviewsForHotel retrieves all reviews for a hotel
func (s *ReviewCrawlerService) GetReviewsForHotel(ctx context.Context, hotelID string) ([]*models.HotelReview, error) {
	query := `
		SELECT id, hotel_id, source, source_review_id, reviewer_name, reviewer_location,
		       rating, review_text, review_date, verified, helpful_count,
		       room_type, travel_type, stay_date, created_at, updated_at
		FROM hotel_reviews
		WHERE hotel_id = ?
		ORDER BY review_date DESC, created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, hotelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*models.HotelReview
	for rows.Next() {
		var review models.HotelReview
		var reviewDate, stayDate sql.NullString
		var rating sql.NullFloat64
		var verified int

		err := rows.Scan(
			&review.ID,
			&review.HotelID,
			&review.Source,
			&review.SourceReviewID,
			&review.ReviewerName,
			&review.ReviewerLocation,
			&rating,
			&review.ReviewText,
			&reviewDate,
			&verified,
			&review.HelpfulCount,
			&review.RoomType,
			&review.TravelType,
			&stayDate,
			&review.CreatedAt,
			&review.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if rating.Valid {
			val := rating.Float64
			review.Rating = &val
		}
		if reviewDate.Valid {
			if t, err := time.Parse(time.RFC3339, reviewDate.String); err == nil {
				review.ReviewDate = &t
			}
		}
		if stayDate.Valid {
			if t, err := time.Parse(time.RFC3339, stayDate.String); err == nil {
				review.StayDate = &t
			}
		}
		review.Verified = verified == 1

		reviews = append(reviews, &review)
	}

	return reviews, rows.Err()
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

	crawler := NewReviewCrawlerService(db)

	ctx := context.Background()

	// Get all hotels
	hotelDB := hotelstorage.NewStorage(db)
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
