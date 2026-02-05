package services

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/edamsoft-sre/alpaca/alpaca/database"
	"github.com/edamsoft-sre/alpaca/alpaca/models"
)

// ReviewCrawlerService handles crawling reviews from multiple sources
type ReviewCrawlerService struct {
	db *database.DB
}

// NewReviewCrawlerService creates a new review crawler service
func NewReviewCrawlerService(db *database.DB) *ReviewCrawlerService {
	return &ReviewCrawlerService{db: db}
}

// ReviewSource defines the interface for review sources
type ReviewSource interface {
	GetSourceName() string
	CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error)
}

// CrawlAllSources crawls reviews from all available sources for a hotel
func (s *ReviewCrawlerService) CrawlAllSources(ctx context.Context, hotel *models.Hotel) (int, error) {
	sources := []ReviewSource{
		NewTripadvisorCrawler(),
		NewGoogleCrawler(),
		NewExpediaCrawler(),
		NewBookingCrawler(),
		NewHotelWebsiteCrawler(),
		NewBingCrawler(),
		NewYelpCrawler(),
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

// TripadvisorCrawler crawls reviews from Tripadvisor
type TripadvisorCrawler struct{}

func NewTripadvisorCrawler() *TripadvisorCrawler {
	return &TripadvisorCrawler{}
}

func (c *TripadvisorCrawler) GetSourceName() string {
	return models.SourceTripadvisor
}

func (c *TripadvisorCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Tripadvisor web scraping or API integration
	log.Printf("Tripadvisor crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}

// GoogleCrawler crawls reviews from Google
type GoogleCrawler struct{}

func NewGoogleCrawler() *GoogleCrawler {
	return &GoogleCrawler{}
}

func (c *GoogleCrawler) GetSourceName() string {
	return models.SourceGoogle
}

func (c *GoogleCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Google Places API integration
	log.Printf("Google crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}

// ExpediaCrawler crawls reviews from Expedia
type ExpediaCrawler struct{}

func NewExpediaCrawler() *ExpediaCrawler {
	return &ExpediaCrawler{}
}

func (c *ExpediaCrawler) GetSourceName() string {
	return models.SourceExpedia
}

func (c *ExpediaCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Expedia API or web scraping
	log.Printf("Expedia crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}

// BookingCrawler crawls reviews from Booking.com
type BookingCrawler struct{}

func NewBookingCrawler() *BookingCrawler {
	return &BookingCrawler{}
}

func (c *BookingCrawler) GetSourceName() string {
	return models.SourceBooking
}

func (c *BookingCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Booking.com API or web scraping
	log.Printf("Booking crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}

// HotelWebsiteCrawler crawls reviews from hotel's own website
type HotelWebsiteCrawler struct{}

func NewHotelWebsiteCrawler() *HotelWebsiteCrawler {
	return &HotelWebsiteCrawler{}
}

func (c *HotelWebsiteCrawler) GetSourceName() string {
	return models.SourceHotelWebsite
}

func (c *HotelWebsiteCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	if hotel.Website == "" {
		return []*models.HotelReview{}, nil
	}

	log.Printf("Hotel website crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}

// BingCrawler crawls reviews from Bing
type BingCrawler struct{}

func NewBingCrawler() *BingCrawler {
	return &BingCrawler{}
}

func (c *BingCrawler) GetSourceName() string {
	return models.SourceBing
}

func (c *BingCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Bing search API integration
	log.Printf("Bing crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}

// YelpCrawler crawls reviews from Yelp
type YelpCrawler struct{}

func NewYelpCrawler() *YelpCrawler {
	return &YelpCrawler{}
}

func (c *YelpCrawler) GetSourceName() string {
	return models.SourceYelp
}

func (c *YelpCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Yelp Fusion API integration
	log.Printf("Yelp crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}
