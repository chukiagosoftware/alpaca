package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chukiagosoftware/alpaca/models"
	"gorm.io/gorm"
)

type googlePlaceDetailsResponse struct {
	// GET https://places.googleapis.com/v1/{name=places/*/photos/*/media} ->
	// returns "name": string,
	//         "photoUri": string
	Photos []struct {
		Name string `json:"name"`
	}
	Reviews []struct {
		AuthorAttribution struct {
			DisplayName string `json:"displayName"`
			URI         string `json:"uri"`
		} `json:"authorAttribution"`
		Rating float64 `json:"rating"`
		Text   struct {
			Text string `json:"text"`
		} `json:"text"`
		PublishTime   string `json:"publishTime"`
		GoogleMapsURI string `json:"googleMapsUri"`
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

func (c *GoogleCrawler) FetchReviewsForLocation(ctx context.Context, locationID string, db *gorm.DB) ([]*models.HotelReview, error) {

	var review models.HotelReview
	err := db.Where("hotel_id = ? AND source = ?", locationID, models.SourceGoogle).
		First(&review).Error

	if err == nil {
		return nil, errors.New(fmt.Sprintf("already fetched reviews for %s", locationID))
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	url := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", locationID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", "reviews,photos")

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

		hash := sha256.New()
		hash.Write([]byte(rev.Text.Text))
		reviewId := hex.EncodeToString(hash.Sum(nil))

		rating := rev.Rating
		var photo string
		if photos := len(detailsResp.Photos); photos > 0 {
			photo = detailsResp.Photos[0].Name
		}

		review := &models.HotelReview{
			ID:               0,
			HotelID:          locationID,
			Source:           models.SourceGoogle,
			SourceReviewID:   reviewId,
			ReviewerName:     rev.AuthorAttribution.DisplayName,
			ReviewerLocation: rev.AuthorAttribution.URI,
			Rating:           rating,
			ReviewText:       rev.Text.Text,
			ReviewDate:       reviewDate,
			Verified:         true,
			GoogleMapsURI:    rev.GoogleMapsURI,
			Photo:            photo,
		}
		reviews = append(reviews, review)
	}

	log.Printf("Fetched %d reviews from Google for hotel %s", len(reviews), locationID)
	log.Println(reviews)
	return reviews, nil
}
