package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chukiagosoftware/alpaca/models"
)

type googlePlaceDetailsResponse struct {
	Reviews []struct {
		AuthorAttribution struct {
			DisplayName string `json:"displayName"`
			URI         string `json:"uri"`
		} `json:"authorAttribution"`
		Rating float64 `json:"rating"`
		Text   struct {
			Text string `json:"text"`
		} `json:"text"`
		PublishTime string `json:"publishTime"`
	} `json:"reviews"`
}

// GoogleCrawler crawls reviews from Google
type GoogleCrawler struct {
	apiKey string
	client *http.Client
}

func NewGoogleCrawler() *GoogleCrawler {
	return &GoogleCrawler{
		apiKey: os.Getenv("GOOGLE_PLACES_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GoogleCrawler) GetSourceName() string {
	return models.SourceGoogle
}

func (c *GoogleCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// Only crawl if the hotel is from Google and has a place ID
	if hotel.Source != models.HotelSourceGoogle || hotel.SourceHotelID == "" {
		log.Printf("Skipping Google reviews for hotel %s (no Google place ID)", hotel.Name)
		return []*models.HotelReview{}, nil
	}

	if c.apiKey == "" {
		log.Printf("Google Places API key not set, skipping reviews for hotel %s", hotel.Name)
		return []*models.HotelReview{}, nil
	}

	url := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", hotel.SourceHotelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", "reviews")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var detailsResp googlePlaceDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&detailsResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var reviews []*models.HotelReview
	for _, rev := range detailsResp.Reviews {
		reviewDate, err := time.Parse(time.RFC3339, rev.PublishTime)
		if err != nil {
			log.Printf("Error parsing review date %s: %v", rev.PublishTime, err)
			continue
		}

		rating := rev.Rating
		review := &models.HotelReview{
			HotelID:        hotel.HotelID,
			Source:         models.SourceGoogle,
			SourceReviewID: rev.PublishTime, // Use publish time as unique ID
			ReviewerName:   rev.AuthorAttribution.DisplayName,
			Rating:         &rating,
			ReviewText:     rev.Text.Text,
			ReviewDate:     &reviewDate,
			Verified:       true, // Google reviews are verified
			HelpfulCount:   0,    // Not provided by API
		}
		reviews = append(reviews, review)
	}

	log.Printf("Fetched %d reviews from Google for hotel %s", len(reviews), hotel.Name)
	return reviews, nil
}
