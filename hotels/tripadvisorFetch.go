package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/chukiagosoftware/alpaca/models"
)

type tripAdvisorSearchResponse struct {
	Data []struct {
		LocationID string `json:"location_id"`
		Name       string `json:"name"`
		AddressObj struct {
			Street1    string `json:"street1"`
			City       string `json:"city"`
			Country    string `json:"country"`
			PostalCode string `json:"postalcode"`
		} `json:"address_obj"`
		Latitude   string  `json:"latitude"`
		Longitude  string  `json:"longitude"`
		Rating     float64 `json:"rating"`
		NumReviews int     `json:"num_reviews"`
		Phone      string  `json:"phone"`
		Website    string  `json:"website"`
		Email      string  `json:"email"`
	} `json:"data"`
}

type tripAdvisorProvider struct {
	apiKey string
	client *http.Client
}

func newTripAdvisorProvider() *tripAdvisorProvider {
	return &tripAdvisorProvider{
		apiKey: os.Getenv("TRIPADVISOR_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *tripAdvisorProvider) getProviderName() string {
	return models.HotelSourceTripadvisor
}

func (p *tripAdvisorProvider) isEnabled() bool {
	return p.apiKey != ""
}

func (p *tripAdvisorProvider) fetchHotels(ctx context.Context, location string) ([]*models.Hotel, error) {
	baseURL := "https://api.content.tripadvisor.com/api/v1/location/search"

	params := url.Values{}
	params.Set("searchQuery", location) //+" hotels"
	params.Set("category", "hotels")
	params.Set("language", "en")
	params.Set("key", p.apiKey)

	requestURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var taResp tripAdvisorSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&taResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var hotels []*models.Hotel
	for _, result := range taResp.Data {
		var lat, lng float64
		fmt.Sscanf(result.Latitude, "%f", &lat)
		fmt.Sscanf(result.Longitude, "%f", &lng)

		rating := result.Rating

		hotel := &models.Hotel{
			HotelID:           fmt.Sprintf("ta_%s", result.LocationID),
			Source:            models.HotelSourceTripadvisor,
			SourceHotelID:     result.LocationID,
			Name:              result.Name,
			City:              result.AddressObj.City,
			Country:           result.AddressObj.Country,
			StreetAddress:     result.AddressObj.Street1,
			PostalCode:        result.AddressObj.PostalCode,
			StateCode:         "",
			Latitude:          lat,
			Longitude:         lng,
			Phone:             result.Phone,
			Website:           result.Website,
			Email:             result.Email,
			TripadvisorRating: rating,
			LastUpdate:        time.Now().Format(time.RFC3339),
		}
		hotels = append(hotels, hotel)
	}

	log.Printf("Fetched %d hotels from TripAdvisor", len(hotels))
	return hotels, nil
}
