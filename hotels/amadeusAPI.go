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
	"strconv"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/hotelstorage"
	"github.com/chukiagosoftware/alpaca/models"
)

// hotelAPIProvider defines the interface for hotel API providers
type hotelAPIProvider interface {
	getOAuthToken(ctx context.Context) (string, error)
	fetchHotelsList(ctx context.Context, cityCode string, token string) ([]models.HotelAPIItem, string, error)
	fetchHotelSearchData(ctx context.Context, hotelID string, token string) (*models.HotelSearchData, error)
	fetchHotelRatingsData(ctx context.Context, hotelID string, token string) (*models.HotelRatingsData, error)
}

// amadeusProvider implements hotelAPIProvider for Amadeus API
type amadeusProvider struct {
	clientID     string
	clientSecret string
	baseURL      string
}

// newAmadeusProvider creates a new Amadeus API provider
func newAmadeusProvider(clientID, clientSecret string) *amadeusProvider {
	return &amadeusProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      "https://test.api.amadeus.com",
	}
}

// getOAuthToken retrieves an OAuth2 token from Amadeus
func (p *amadeusProvider) getOAuthToken(ctx context.Context) (string, error) {
	baseURL := p.baseURL + "/v1/security/oauth2/token"
	data := url.Values{}
	data.Set("client_secret", p.clientSecret)
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get token, status code: %d: %s", resp.StatusCode, string(body))
	}

	var token models.HotelAmadeusOauth2
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}

	log.Println("OAuth2 token obtained successfully")
	return token.AccessToken, nil
}

// fetchHotelsList fetches hotels list with pagination
func (p *amadeusProvider) fetchHotelsList(ctx context.Context, cityCode string, token string) ([]models.HotelAPIItem, string, error) {
	hotelListURL := os.Getenv("AMADEUS_HOTEL_LIST_URL")
	if hotelListURL == "" {
		hotelListURL = p.baseURL + "/v1/reference-data/locations/hotels/by-city"
	}

	radius := os.Getenv("HOTEL_SEARCH_RADIUS")
	if radius == "" {
		radius = "5" // Default radius
	}

	radiusUnit := os.Getenv("HOTEL_SEARCH_RADIUS_UNIT")
	if radiusUnit == "" {
		radiusUnit = "KM" // Default unit
	}

	data := url.Values{}
	data.Set("cityCode", cityCode)
	data.Set("radius", radius)
	data.Set("radiusUnit", radiusUnit)

	var allHotels []models.HotelAPIItem
	currentURL := hotelListURL + "?" + data.Encode()
	page := 1

	for {
		req, err := http.NewRequestWithContext(ctx, "GET", currentURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		log.Printf("Fetching page %d for cityCode: %s", page, cityCode)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("error making request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var apiResp models.HotelsListResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, "", fmt.Errorf("error decoding response: %w", err)
		}

		allHotels = append(allHotels, apiResp.Data...)

		// Check if there's a next page
		if apiResp.Meta.Links.Next == "" {
			log.Printf("No next page available, stopping pagination")
			break
		}

		currentURL = apiResp.Meta.Links.Next
		page++

		// Rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Successfully fetched %d hotels total", len(allHotels))
	return allHotels, "", nil
}

// fetchHotelSearchData fetches detailed hotel search data
func (p *amadeusProvider) fetchHotelSearchData(ctx context.Context, hotelID string, token string) (*models.HotelSearchData, error) {
	hotelSearchURL := os.Getenv("AMADEUS_HOTEL_SEARCH_URL")
	if hotelSearchURL == "" {
		hotelSearchURL = p.baseURL + "/v2/shopping/hotel-offers"
	}

	data := url.Values{}
	data.Set("hotelIds", hotelID)
	requestURL := hotelSearchURL + "?" + data.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error status %d for hotel %s search", resp.StatusCode, hotelID)
	}

	var searchResp models.HotelSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	if len(searchResp.Data) > 0 {
		return &searchResp.Data[0], nil
	}

	return nil, fmt.Errorf("no data returned for hotel %s", hotelID)
}

// fetchHotelRatingsData fetches hotel ratings data
func (p *amadeusProvider) fetchHotelRatingsData(ctx context.Context, hotelID string, token string) (*models.HotelRatingsData, error) {
	hotelRatingsURL := os.Getenv("AMADEUS_HOTEL_RATINGS_URL")
	if hotelRatingsURL == "" {
		hotelRatingsURL = p.baseURL + "/v2/e-reputation/hotel-sentiments"
	}

	requestURL := hotelRatingsURL + "?hotelIds=" + hotelID

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error status %d for hotel %s ratings", resp.StatusCode, hotelID)
	}

	var ratingsResp models.HotelRatingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&ratingsResp); err != nil {
		return nil, err
	}

	if len(ratingsResp.Data) > 0 {
		return &ratingsResp.Data[0], nil
	}

	return nil, fmt.Errorf("no data returned for hotel %s", hotelID)
}

// fetchHotelsForCity fetches hotels from Amadeus for a given city code
func fetchHotelsForCity(ctx context.Context, hotelStorage *hotelstorage.Storage, cityCode string) (int, error) {
	apiClient := os.Getenv("AMD")
	apiSecret := os.Getenv("AMS")

	if apiClient == "" || apiSecret == "" {
		return 0, fmt.Errorf("Amadeus credentials not provided")
	}

	provider := newAmadeusProvider(apiClient, apiSecret)

	token, err := provider.getOAuthToken(ctx)
	if err != nil {
		return 0, fmt.Errorf("error getting OAuth token: %w", err)
	}

	hotels, _, err := provider.fetchHotelsList(ctx, cityCode, token)
	if err != nil {
		return 0, fmt.Errorf("error fetching hotels list: %w", err)
	}

	// Process hotels
	hotelsCreated := 0
	for _, hotel := range hotels {
		err := hotelStorage.Create(ctx, &hotel)
		if err != nil {
			log.Printf("Error saving hotel %s: %v", hotel.Name, err)
		} else {
			hotelsCreated++
		}
	}

	return hotelsCreated, nil
}
