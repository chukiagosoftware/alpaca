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

type expediaSearchResponse struct {
	Properties []struct {
		PropertyID string `json:"property_id"`
		Name       string `json:"name"`
		Address    struct {
			Line1       string `json:"line_1"`
			City        string `json:"city"`
			StateCode   string `json:"state_province_code"`
			PostalCode  string `json:"postal_code"`
			CountryCode string `json:"country_code"`
		} `json:"address"`
		Location struct {
			Coordinates struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"coordinates"`
		} `json:"location"`
		Ratings struct {
			Property struct {
				Rating float64 `json:"rating"`
			} `json:"property"`
		} `json:"ratings"`
		Phone string `json:"phone"`
	} `json:"properties"`
}

type expediaProvider struct {
	apiKey string
	client *http.Client
}

func newExpediaProvider() *expediaProvider {
	return &expediaProvider{
		apiKey: os.Getenv("EXPEDIA_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *expediaProvider) getProviderName() string {
	return models.HotelSourceExpedia
}

func (p *expediaProvider) isEnabled() bool {
	return p.apiKey != ""
}

func (p *expediaProvider) fetchHotels(ctx context.Context, location string) ([]*models.Hotel, error) {
	// Expedia Rapid API search endpoint
	baseURL := "https://api.ean.com/2.3/properties/search"

	params := url.Values{}
	params.Set("location", location)
	params.Set("sort", "recommended")
	params.Set("limit", "100")

	requestURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
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

	var expediaResp expediaSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&expediaResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var hotels []*models.Hotel
	for _, prop := range expediaResp.Properties {
		lat := prop.Location.Coordinates.Latitude
		lng := prop.Location.Coordinates.Longitude
		rating := prop.Ratings.Property.Rating

		hotel := &models.Hotel{
			HotelID:       fmt.Sprintf("expedia_%s", prop.PropertyID),
			Source:        models.HotelSourceExpedia,
			SourceHotelID: prop.PropertyID,
			Name:          prop.Name,
			City:          prop.Address.City,
			Country:       prop.Address.CountryCode,
			StreetAddress: prop.Address.Line1,
			PostalCode:    prop.Address.PostalCode,
			StateCode:     prop.Address.StateCode,
			Latitude:      &lat,
			Longitude:     &lng,
			Phone:         prop.Phone,
			ExpediaRating: &rating,
			LastUpdate:    time.Now().Format(time.RFC3339),
		}
		hotels = append(hotels, hotel)
	}

	log.Printf("Fetched %d hotels from Expedia", len(hotels))
	return hotels, nil
}
