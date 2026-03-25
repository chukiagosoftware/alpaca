package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/models"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// TripAdvisorReviewsService handles fetching reviews from TripAdvisor Content API
type TripAdvisorReviewsService struct {
	client  *resty.Client
	apiKey  string
	baseURL string
}

func (s *TripAdvisorReviewsService) GetSourceName() string {
	return "tripadvisorAPI"

}

// NewTripAdvisorReviewsService initializes the service with partner API key
func NewTripAdvisorReviewsService() *TripAdvisorReviewsService {
	godotenv.Load()
	apiKey := os.Getenv("TRIPADVISOR_API_KEY")
	if apiKey == "" {
		log.Fatal("Missing TRIPADVISOR_API_KEY for Content API")
	}

	rand.Seed(time.Now().UnixNano()) // Seed for random delays

	client := resty.New().
		SetTimeout(15 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &TripAdvisorReviewsService{
		client:  client,
		apiKey:  apiKey,
		baseURL: "https://api.content.tripadvisor.com/api/v1",
	}
}

func parseTripAdvisorDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	dateStr = strings.TrimSpace(dateStr)

	// Common TripAdvisor layouts
	layouts := []string{
		time.RFC3339,           // "2023-10-15T12:00:00Z"
		"2006-01-02T15:04:05Z", // ISO without full RFC
		"2006-01-02",           // "2023-10-15" (simple date)
		"Jan 2, 2006",          // "October 15, 2023"
		"2006-01-02 15:04:05",  // With time, no TZ
		"Jan _2 2006",          // "Oct 15 2023" (no comma)
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t, nil
		}
	}

	// Fallback for relative dates ("2 months ago", "1 week ago")
	if strings.HasSuffix(strings.ToLower(dateStr), "ago") {
		parts := strings.Fields(dateStr)
		if len(parts) >= 2 {
			num, err := strconv.Atoi(parts[0])
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid relative date: %s", dateStr)
			}
			unit := strings.ToLower(parts[1])

			now := time.Now().UTC() // Assume UTC for consistency
			switch {
			case strings.HasPrefix(unit, "day"):
				return now.AddDate(0, 0, -num), nil
			case strings.HasPrefix(unit, "week"):
				return now.AddDate(0, 0, -num*7), nil
			case strings.HasPrefix(unit, "month"):
				return now.AddDate(0, -num, 0), nil
			case strings.HasPrefix(unit, "year"):
				return now.AddDate(-num, 0, 0), nil
			}
		}
	}

	// Log failure (non-fatal)
	log.Printf("Failed to parse TripAdvisor date '%s' (tried layouts: %v)", dateStr, layouts)
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", dateStr)
}

func (s *TripAdvisorReviewsService) randomDelay(minMs, maxMs int) {
	delayMs := rand.Intn(maxMs-minMs+1) + minMs
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}

// FetchReviewsForLocation fetches reviews for a TripAdvisor locationID (e.g., hotel)
// Endpoint: /location/{location_id}/reviews
// Simple offset and break based on len(apiResp.Data) an array of reviews. Pagination not available for basic API
func (s *TripAdvisorReviewsService) FetchReviewsForLocation(ctx context.Context, locationID string, db *gorm.DB) ([]*models.HotelReview, error) {

	var review models.HotelReview
	err := db.Where("hotel_id = ? AND source = ?", locationID, models.SourceTripadvisor).
		First(&review).Error

	if err == nil {
		return nil, errors.New(fmt.Sprintf("already fetched reviews for %s", locationID))
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	maxReviews := 100
	var allReviews []*models.HotelReview
	offset := 0

	for len(allReviews) < maxReviews {
		url := fmt.Sprintf("%s/location/%s/reviews", s.baseURL, locationID)

		resp, err := s.client.R().
			SetContext(ctx).
			SetQueryParams(map[string]string{
				"key":          s.apiKey,
				"language":     "en",
				"reviews_sort": "recent_first",
				"limit":        strconv.Itoa(maxReviews),
				"offset":       strconv.Itoa(offset),
			}).
			Get(url)
		if err != nil {
			return nil, fmt.Errorf("API request failed for location %s (offset %d): %w", locationID, offset, err)
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("API error for location %s (offset %d): %s - %s", locationID, offset, resp.Status(), resp.String())
		}

		// Parse response (your struct; added fallback for "total" if "total_results" wrong)
		var apiResp struct {
			Data []struct {
				ID          int32   `json:"id"`
				Rating      float64 `json:"rating"`
				HelpfulVote int32   `json:"helpful_votes"`
				Text        string  `json:"text"`
				Title       string  `json:"title"`
				ReviewDate  string  `json:"published_date"`
				User        struct {
					Username     string `json:"username"`
					ReviewCount  int32  `json:"review_count"`
					UserLocation struct {
						Name string `json:"user_location"`
						Id   string `json:"id"`
					}
				}
				OwnerResponse struct {
					Text   string `json:"text"`
					Title  string `json:"title"`
					Author string `json:"author"`
				} `json:"owner_response"`
			} `json:"data"`
			Paging struct {
				Next         string `json:"next"`
				Previous     string `json:"previous"`
				Results      int    `json:"results"`
				Skipped      int    `json:"skipped"`
				TotalResults int    `json:"total_results"`
			} `json:"paging"`
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    int    `json:"code"`
			}
		}
		if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
			return nil, fmt.Errorf("JSON unmarshal failed for location %s (offset %d): %w", locationID, offset, err)
		}

		reviews := apiResp.Data
		if len(reviews) == 0 {
			log.Printf("No more reviews received for location %s (offset %d)", locationID, offset)
			break
		}

		for _, r := range reviews {
			hash := sha256.New()
			// identify matching review text instead of arbitrary ID across method
			hash.Write([]byte(r.Text))
			sourceReviewID := hex.EncodeToString(hash.Sum(nil))
			//sourceReviewID := int32(r.ID)

			date, dateErr := parseTripAdvisorDate(r.ReviewDate)
			if dateErr != nil {
				log.Printf("Skipping review %d due to date parse error: %v", sourceReviewID, dateErr)
				continue
			}

			allReviews = append(allReviews, &models.HotelReview{
				// Let Gorm set record ID
				HotelID:          locationID,
				Source:           models.SourceTripadvisor,
				SourceReviewID:   sourceReviewID,
				ReviewText:       r.Title + "\n" + r.Text,
				Rating:           r.Rating,
				ReviewDate:       date,
				ReviewerName:     r.User.Username,
				ReviewerLocation: r.User.UserLocation.Name,
				HelpfulCount:     int(r.HelpfulVote),
			})
		}

		// Random delay
		s.randomDelay(300, 800)

		log.Printf("Received: %d reviews for location %s, offset:%d\n", len(reviews), locationID, offset)

		offset += len(reviews)
	}

	return allReviews, nil
}
