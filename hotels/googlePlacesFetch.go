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

type googlePlacesResponse struct {
	Results []struct {
		PlaceID          string `json:"place_id"`
		Name             string `json:"name"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		Rating           float64 `json:"rating"`
		UserRatingsTotal int     `json:"user_ratings_total"`
	} `json:"results"`
	NextPageToken string `json:"next_page_token"`
	Status        string `json:"status"`
}

type googlePlacesProvider struct {
	apiKey string
	client *http.Client
}

func newGooglePlacesProvider() *googlePlacesProvider {
	return &googlePlacesProvider{
		apiKey: os.Getenv("GOOGLE_PLACES_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *googlePlacesProvider) getProviderName() string {
	return models.HotelSourceGoogle
}

func (p *googlePlacesProvider) isEnabled() bool {
	return p.apiKey != ""
}

func (p *googlePlacesProvider) fetchHotels(ctx context.Context, location string) ([]*models.Hotel, error) {
	baseURL := "https://maps.googleapis.com/maps/api/place/textsearch/json"
	params := url.Values{}
	params.Set("query", fmt.Sprintf("hotels in %s", location))
	params.Set("key", p.apiKey)
	params.Set("type", "lodging")

	var allHotels []*models.Hotel
	nextPageToken := ""

	for page := 0; page < 3; page++ { // Limit to 3 pages (60 results)
		requestURL := baseURL + "?" + params.Encode()
		if nextPageToken != "" {
			requestURL += "&pagetoken=" + nextPageToken
		}

		req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var placesResp googlePlacesResponse
		if err := json.NewDecoder(resp.Body).Decode(&placesResp); err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		if placesResp.Status != "OK" && placesResp.Status != "ZERO_RESULTS" {
			return nil, fmt.Errorf("API returned status: %s", placesResp.Status)
		}

		for _, result := range placesResp.Results {
			lat := result.Geometry.Location.Lat
			lng := result.Geometry.Location.Lng
			rating := result.Rating

			hotel := &models.Hotel{
				HotelID:       fmt.Sprintf("google_%s", result.PlaceID),
				Source:        models.HotelSourceGoogle,
				SourceHotelID: result.PlaceID,
				Name:          result.Name,
				StreetAddress: result.FormattedAddress,
				StateCode:     parseStateFromAddress(result.FormattedAddress),
				Latitude:      &lat,
				Longitude:     &lng,
				GoogleRating:  &rating,
				LastUpdate:    time.Now().Format(time.RFC3339),
			}
			allHotels = append(allHotels, hotel)
		}

		log.Printf("Fetched page %d from Google: %d hotels (total: %d)", page+1, len(placesResp.Results), len(allHotels))

		if placesResp.NextPageToken == "" {
			break
		}
		nextPageToken = placesResp.NextPageToken
		time.Sleep(2 * time.Second) // Google requires delay
	}

	return allHotels, nil
}
