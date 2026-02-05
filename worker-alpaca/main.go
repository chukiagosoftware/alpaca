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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edamsoft-sre/alpaca/database"
	"github.com/edamsoft-sre/alpaca/models"
	"github.com/edamsoft-sre/alpaca/services"
	"github.com/edamsoft-sre/alpaca/utils"
	"github.com/joho/godotenv"
)

// HotelAPIProvider defines the interface for hotel API providers
type HotelAPIProvider interface {
	GetOAuthToken(ctx context.Context) (string, error)
	FetchHotelsList(ctx context.Context, cityCode string, token string) ([]models.HotelAPIItem, string, error)
	FetchHotelSearchData(ctx context.Context, hotelID string, token string) (*models.HotelSearchData, error)
	FetchHotelRatingsData(ctx context.Context, hotelID string, token string) (*models.HotelRatingsData, error)
}

// AmadeusProvider implements HotelAPIProvider for Amadeus API
type AmadeusProvider struct {
	clientID     string
	clientSecret string
	baseURL      string
}

// NewAmadeusProvider creates a new Amadeus API provider
func NewAmadeusProvider(clientID, clientSecret string) *AmadeusProvider {
	return &AmadeusProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      "https://test.api.amadeus.com",
	}
}

// GetOAuthToken retrieves an OAuth2 token from Amadeus
func (p *AmadeusProvider) GetOAuthToken(ctx context.Context) (string, error) {
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

// FetchHotelsList fetches hotels list with pagination
func (p *AmadeusProvider) FetchHotelsList(ctx context.Context, cityCode string, token string) ([]models.HotelAPIItem, string, error) {
	hotelListURL := os.Getenv("AMADEUS_HOTEL_LIST_URL")
	if hotelListURL == "" {
		hotelListURL = p.baseURL + "/v1/reference-data/locations/hotels/by-city"
	}

	radius := os.Getenv(utils.HotelSearchRadius)
	if radius == "" {
		radius = utils.DefaultRadius
	}

	radiusUnit := os.Getenv(utils.HotelSearchRadiusUnit)
	if radiusUnit == "" {
		radiusUnit = utils.DefaultRadiusUnit
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

// FetchHotelSearchData fetches detailed hotel search data
func (p *AmadeusProvider) FetchHotelSearchData(ctx context.Context, hotelID string, token string) (*models.HotelSearchData, error) {
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

// FetchHotelRatingsData fetches hotel ratings data
func (p *AmadeusProvider) FetchHotelRatingsData(ctx context.Context, hotelID string, token string) (*models.HotelRatingsData, error) {
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

// fetchHotelsListPaginated fetches all hotels with proper pagination handling
func fetchHotelsListPaginated(ctx context.Context, provider HotelAPIProvider, hotelService *services.HotelService, cityCode string) (int, error) {
	hotelsCreated := 0

	token, err := provider.GetOAuthToken(ctx)
	if err != nil {
		return 0, fmt.Errorf("error getting OAuth token: %w", err)
	}

	hotels, _, err := provider.FetchHotelsList(ctx, cityCode, token)
	if err != nil {
		return 0, fmt.Errorf("error fetching hotels list: %w", err)
	}

	// Process hotels
	for _, hotel := range hotels {
		err := hotelService.Create(ctx, &hotel)
		if err != nil {
			log.Printf("Error saving hotel %s: %v", hotel.Name, err)
		} else {
			log.Printf("Processing: %s", hotel.Name)
			hotelsCreated++
		}
	}

	log.Printf("Successfully fetched %d hotels total", hotelsCreated)
	return hotelsCreated, nil
}

// fetchHotelSearchData fetches detailed hotel information for each hotel ID
func fetchHotelSearchData(ctx context.Context, provider HotelAPIProvider, hotelService *services.HotelService, hotelIDs []string) (int, error) {
	searchDataCreated := 0
	semaphore := make(chan struct{}, 5) // Limit concurrent requests
	var wg sync.WaitGroup
	var mu sync.Mutex

	log.Printf("Starting to fetch search data for %d hotels", len(hotelIDs))

	token, err := provider.GetOAuthToken(ctx)
	if err != nil {
		return 0, fmt.Errorf("error getting OAuth token: %w", err)
	}

	for _, hotelID := range hotelIDs {
		wg.Add(1)
		go func(hotelID string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if hotel ID is invalid
			invalid, err := hotelService.IsHotelIDInvalidForSearch(ctx, hotelID)
			if err != nil {
				log.Printf("Error checking invalid hotel ID %s: %v", hotelID, err)
				return
			}
			if invalid {
				log.Printf("Skipping invalid hotel ID: %s", hotelID)
				return
			}

			hotelData, err := provider.FetchHotelSearchData(ctx, hotelID, token)
			if err != nil {
				log.Printf("Error fetching search data for hotel %s: %v", hotelID, err)
				// Mark as invalid if it's a persistent error
				if err := hotelService.MarkHotelIDInvalidForSearch(ctx, hotelID); err != nil {
					log.Printf("Error marking hotel ID as invalid: %v", err)
				}
				return
			}

			err = hotelService.UpsertSearchData(ctx, hotelData)
			if err != nil {
				log.Printf("Error saving search data for hotel %s: %v", hotelID, err)
			} else {
				log.Printf("Fetched and saved search data for hotel: %s", hotelData.Name)

				mu.Lock()
				searchDataCreated++
				mu.Unlock()
			}

			// Rate limiting
			time.Sleep(200 * time.Millisecond)
		}(hotelID)
	}

	wg.Wait()
	log.Printf("Successfully fetched search data for %d hotels", searchDataCreated)
	return searchDataCreated, nil
}

// fetchHotelRatingsData fetches ratings and sentiment data for each hotel ID
func fetchHotelRatingsData(ctx context.Context, provider HotelAPIProvider, hotelService *services.HotelService, hotelIDs []string) (int, error) {
	ratingsCreated := 0
	semaphore := make(chan struct{}, 1) // Limit concurrent requests
	var wg sync.WaitGroup
	var mu sync.Mutex

	log.Printf("Starting to fetch ratings data for %d hotels", len(hotelIDs))

	token, err := provider.GetOAuthToken(ctx)
	if err != nil {
		return 0, fmt.Errorf("error getting OAuth token: %w", err)
	}

	for _, hotelID := range hotelIDs {
		wg.Add(1)
		go func(hotelID string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ratingData, err := provider.FetchHotelRatingsData(ctx, hotelID, token)
			if err != nil {
				log.Printf("Error fetching ratings data for hotel %s: %v", hotelID, err)
				return
			}

			err = hotelService.UpsertRatingsData(ctx, ratingData)
			if err != nil {
				log.Printf("Error saving ratings data for hotel %s: %v", hotelID, err)
			} else {
				log.Printf("Fetched and saved ratings data for hotel: %s (Rating: %d)", hotelID, ratingData.OverallRating)

				mu.Lock()
				ratingsCreated++
				mu.Unlock()
			}

			// Rate limiting
			time.Sleep(200 * time.Millisecond)
		}(hotelID)
	}

	wg.Wait()
	log.Printf("Successfully fetched ratings data for %d hotels", ratingsCreated)
	return ratingsCreated, nil
}

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(currentFile))
	_ = godotenv.Load(filepath.Join(projectRoot, ".env"))

	// Generate top cities data (uncomment to regenerate)
	if err := GenerateTopCities(); err != nil {
		log.Printf("Warning: Failed to generate top cities: %v", err)
	}

	apiClient := os.Getenv("AMD")
	apiSecret := os.Getenv("AMS")

	if apiClient == "" || apiSecret == "" {
		log.Fatal("AMD and AMS environment variables are required")
	}

	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize services
	hotelService := services.NewHotelService(db)

	// Initialize API provider
	provider := NewAmadeusProvider(apiClient, apiSecret)

	ctx := context.Background()

	// Step 1: Fetch hotel list using V1 API
	log.Println("=== Step 1: Fetching hotel list ===")
	cityCode := "AUS" // Default to Austin, can be made configurable
	hotelsCreated, err := fetchHotelsListPaginated(ctx, provider, hotelService, cityCode)
	if err != nil {
		log.Printf("Error fetching hotel list: %v", err)
	} else {
		log.Printf("Successfully fetched %d hotels total", hotelsCreated)
	}

	// Step 2: Get hotel IDs for further processing
	log.Println("=== Step 2: Getting hotel IDs ===")
	hotelIDs, err := hotelService.GetHotelIDs(ctx)
	if err != nil {
		log.Printf("Error getting hotel IDs: %v", err)
		return
	}
	log.Printf("Retrieved %d hotel IDs for processing", len(hotelIDs))

	// Step 3: Fetch hotel search data using V2 API
	log.Println("=== Step 3: Fetching hotel search data ===")
	searchCreated, err := fetchHotelSearchData(ctx, provider, hotelService, hotelIDs)
	if err != nil {
		log.Printf("Error fetching hotel search data: %v", err)
	} else {
		log.Printf("Successfully fetched search data for %d hotels", searchCreated)
	}

	// Step 4: Fetch hotel ratings data using V2 API (using test hotel IDs for now)
	log.Println("=== Step 4: Fetching hotel ratings data ===")
	ratingsCreated, err := fetchHotelRatingsData(ctx, provider, hotelService, utils.TestHotelIDs)
	if err != nil {
		log.Printf("Error fetching hotel ratings data: %v", err)
	} else {
		log.Printf("Successfully fetched ratings data for %d hotels", ratingsCreated)
	}

	// Summary
	log.Println("=== Summary ===")
	log.Printf("Hotels fetched: %d", hotelsCreated)
	log.Printf("Search data fetched: %d", searchCreated)
	log.Printf("Ratings data fetched: %d", ratingsCreated)
}
