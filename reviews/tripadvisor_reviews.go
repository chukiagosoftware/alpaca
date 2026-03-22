package main

import (
	"context"
	"encoding/json"
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

// FetchReviewsForLocation fetches reviews for a TripAdvisor location ID (e.g., hotel)
// Endpoint: /location/{location_id}/reviews
// Paginated: Loops with reviews_start_at until maxReviews or no more
// Returns up to maxReviews (default 100); sorts by recent
func (s *TripAdvisorReviewsService) FetchReviewsForLocation(ctx context.Context, locationID string, maxReviews int) ([]*models.HotelReview, error) {
	if maxReviews == 0 {
		maxReviews = 100 // Default
	}

	var allReviews []*models.HotelReview
	offset := 0
	limit := 5     // Free tier; set to 100 for paid
	maxPages := 10 // Safety: Prevent infinite loop if total=0

	pageCount := 0
	//var currentReviewID int32 = 20000

	for len(allReviews) < maxReviews && pageCount < maxPages {
		url := fmt.Sprintf("%s/location/%s/reviews", s.baseURL, locationID)

		resp, err := s.client.R().
			SetContext(ctx).
			SetQueryParams(map[string]string{
				"key":            s.apiKey,
				"lang":           "en_US",
				"currency":       "USD",
				"reviews_sort":   "recent_first",
				"reviews_limit":  strconv.Itoa(limit),
				"reviews_offset": strconv.Itoa(offset),
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
				TotalResults int `json:"total_results"`
				Total        int `json:"total"` // Fallback if API uses "total"
			} `json:"paging"`
		}
		if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
			return nil, fmt.Errorf("JSON unmarshal failed for location %s (offset %d): %w", locationID, offset, err)
		}

		reviews := apiResp.Data
		total := apiResp.Paging.TotalResults
		if total == 0 {
			total = apiResp.Paging.Total // Try fallback
		}

		if len(reviews) == 0 {
			log.Printf("No more reviews for location %s (offset %d, total: %d)", locationID, offset, total)
			break // End of data
		}

		if len(reviews) < limit {
			log.Printf("Partial page for location %s (offset %d): %d < limit %d; end of data", locationID, offset, len(reviews), limit)
			// Don't increment offset—last page
		}

		// Map to models.HotelReview (adjust fields to your model)
		for _, r := range reviews {
			if len(allReviews) >= maxReviews {
				break
			}

			sourceReviewID := int32(r.ID)
			if sourceReviewID == 0 {
				log.Printf("Skipping invalid review ID 0 for location %s", locationID)
				continue
			}

			date, dateErr := parseTripAdvisorDate(r.ReviewDate)
			if dateErr != nil {
				log.Printf("Skipping review %d due to date parse error: %v", sourceReviewID, dateErr)
				continue
			}

			allReviews = append(allReviews, &models.HotelReview{
				ID:               0,          // Let GORM auto-generate primary key
				HotelID:          locationID, // Temporary; overridden in BatchFetchReviewsForHotels
				Source:           models.SourceTripadvisorAPI,
				SourceReviewID:   sourceReviewID,
				ReviewText:       r.Title + "\n" + r.Text,
				Rating:           r.Rating,
				ReviewDate:       date,
				ReviewerName:     r.User.Username,
				ReviewerLocation: r.User.UserLocation.Name,
				HelpfulCount:     int(r.HelpfulVote),
			})
		}

		offset += limit
		pageCount++
		log.Printf("Fetched %d reviews for location %s (offset %d, page %d, total so far: %d / %d)", len(reviews), locationID, offset-limit, pageCount, len(allReviews), total)

		// Random delay
		s.randomDelay(300, 800)

		// Stop if reached total or partial page
		if len(reviews) < limit || offset >= total {
			log.Printf("Pagination complete for location %s (reached end/total %d)", locationID, total)
			break
		}

		if pageCount >= maxPages {
			log.Printf("Warning: Hit max pages (%d) for location %s - possible infinite data or parsing issue", maxPages, locationID)
		}

		log.Printf("Completed: %d reviews for location %s (from %d pages, total available: %d)", len(allReviews), locationID, pageCount, total)
	}
	return allReviews, nil
}

// BatchFetchReviewsForHotels fetches reviews for multiple hotels (e.g., from selected cities)
// Use after fetching hotels with TA location IDs
func (s *TripAdvisorReviewsService) BatchFetchReviewsForHotels(ctx context.Context, hotels []*models.Hotel, maxPerHotel int) map[string][]*models.HotelReview {
	reviewsMap := make(map[string][]*models.HotelReview)
	for _, hotel := range hotels {
		if hotel.SourceHotelID == "" { // Assume field for TripAdvisor ID
			log.Printf("Skipping hotel %s: No TA location ID", hotel.Name)
			continue
		}

		reviews, err := s.FetchReviewsForLocation(ctx, hotel.SourceHotelID, maxPerHotel)
		if err != nil {
			log.Printf("Failed reviews for %s (%s): %v", hotel.Name, hotel.SourceHotelID, err)
			continue
		}
		for _, review := range reviews {
			review.HotelID = hotel.HotelID // Link to internal hotel ID
		}
		reviewsMap[hotel.HotelID] = reviews

		time.Sleep(1 * time.Second) // Rate limit between hotels
	}
	return reviewsMap
}
