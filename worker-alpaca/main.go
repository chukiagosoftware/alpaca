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
	"github.com/joho/godotenv"
)

// fetchHotelsListPaginated fetches all hotels with proper pagination handling using API links
func fetchHotelsListPaginated(ctx context.Context, hotelService *services.HotelService, baseURL, apiToken string) (int, error) {
	hotels_created := 0

	// Build initial URL with required parameters
	data := url.Values{}
	data.Set("cityCode", "AUS")
	currentURL := baseURL + "?" + data.Encode()
	page := 1

	for {
		req, err := http.NewRequestWithContext(ctx, "GET", currentURL, nil)
		if err != nil {
			return hotels_created, fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiToken)

		log.Printf("Fetching page %d for cityCode: AUS", page)
		log.Printf("Request URL: %s", currentURL)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return hotels_created, fmt.Errorf("error making request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return hotels_created, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var apiResp models.HotelsListResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return hotels_created, fmt.Errorf("error decoding response: %w", err)
		}

		// Process hotels in this page
		for _, hotel := range apiResp.Data {
			err := hotelService.Create(ctx, &hotel)
			if err != nil {
				log.Printf("Error saving hotel %s: %v", hotel.Name, err)
			} else {
				log.Printf("Processing: %s", hotel.Name)
				hotels_created++
			}
		}

		// Check if there's a next page
		if apiResp.Meta.Links.Next == "" {
			log.Printf("No next page available, stopping pagination")
			break
		}

		// Use the next page URL from the API response
		currentURL = apiResp.Meta.Links.Next
		page++

		// Rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Successfully fetched %d hotels total", hotels_created)
	return hotels_created, nil
}

// fetchHotelSearchData fetches detailed hotel information for each hotel ID
func fetchHotelSearchData(ctx context.Context, hotelService *services.HotelService, baseURL, apiToken string, hotelIDs []string) (int, error) {
	searchDataCreated := 0
	semaphore := make(chan struct{}, 5) // Limit concurrent requests
	var wg sync.WaitGroup
	var mu sync.Mutex

	log.Printf("Starting to fetch search data for %d hotels", len(hotelIDs))

	for _, hotelID := range hotelIDs {
		wg.Add(1)
		go func(hotelID string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Build request URL for hotel search
			data := url.Values{}
			data.Set("hotelIds", hotelID)
			requestURL := baseURL + "?" + data.Encode()

			req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
			if err != nil {
				log.Printf("Error creating search request for hotel %s: %v", hotelID, err)
				return
			}
			req.Header.Set("Authorization", "Bearer "+apiToken)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Error fetching search data for hotel %s: %v", hotelID, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("Error status %d for hotel %s search", resp.StatusCode, hotelID)
				return
			}

			var searchResp models.HotelSearchResponse
			if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
				log.Printf("Error decoding search response for hotel %s: %v", hotelID, err)
				return
			}

			if len(searchResp.Data) > 0 {
				hotelData := searchResp.Data[0]
				err := hotelService.UpsertSearchData(ctx, &hotelData)
				if err != nil {
					log.Printf("Error saving search data for hotel %s: %v", hotelID, err)
				} else {
					log.Printf("Fetched and saved search data for hotel: %s", hotelData.Name)

					mu.Lock()
					searchDataCreated++
					mu.Unlock()
				}
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
func fetchHotelRatingsData(ctx context.Context, hotelService *services.HotelService, baseURL, apiToken string, hotelIDs []string) (int, error) {
	ratingsCreated := 0
	semaphore := make(chan struct{}, 5) // Limit concurrent requests to match search
	var wg sync.WaitGroup
	var mu sync.Mutex

	log.Printf("Starting to fetch ratings data for %d hotels", len(hotelIDs))

	for _, hotelID := range hotelIDs {
		wg.Add(1)
		go func(hotelID string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Build request URL for hotel ratings
			requestURL := baseURL + "?hotelIds=" + hotelID

			req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
			if err != nil {
				log.Printf("Error creating ratings request for hotel %s: %v", hotelID, err)
				return
			}
			req.Header.Set("Authorization", "Bearer "+apiToken)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Error fetching ratings data for hotel %s: %v", hotelID, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("Error status %d for hotel %s ratings", resp.StatusCode, hotelID)
				return
			}

			var ratingsResp models.HotelRatingsResponse
			if err := json.NewDecoder(resp.Body).Decode(&ratingsResp); err != nil {
				log.Printf("Error decoding ratings response for hotel %s: %v", hotelID, err)
				return
			}

			if len(ratingsResp.Data) > 0 {
				ratingData := ratingsResp.Data[0]
				err := hotelService.UpsertRatingsData(ctx, &ratingData)
				if err != nil {
					log.Printf("Error saving ratings data for hotel %s: %v", hotelID, err)
				} else {
					log.Printf("Fetched and saved ratings data for hotel: %s (Rating: %d)", hotelID, ratingData.OverallRating)

					mu.Lock()
					ratingsCreated++
					mu.Unlock()
				}
			}

			// Rate limiting - match search implementation
			time.Sleep(200 * time.Millisecond)
		}(hotelID)
	}

	wg.Wait()
	log.Printf("Successfully fetched ratings data for %d hotels", ratingsCreated)
	return ratingsCreated, nil
}

// getHotelIDs retrieves all hotel IDs from the database for processing
func getHotelIDs(ctx context.Context, hotelService *services.HotelService) ([]string, error) {
	hotelIDs, err := hotelService.GetHotelIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting hotel IDs: %w", err)
	}

	log.Printf("Retrieved %d hotel IDs for processing", len(hotelIDs))
	return hotelIDs, nil
}

func oauth2_token(ctx context.Context, client_secret, client_id string) (string, error) {
	baseUrl := "https://test.api.amadeus.com/v1/security/oauth2/token"
	data := url.Values{}
	data.Set("client_secret", client_secret)
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", client_id)

	req, _ := http.NewRequestWithContext(ctx, "POST", baseUrl, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token, status code: %d", resp.StatusCode)
	}

	var token models.HotelAmadeusOauth2

	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		log.Fatal(err)
	}
	log.Println("Oauth2 token saved.")
	return token.Access_token, nil
}

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(currentFile))
	_ = godotenv.Load(filepath.Join(projectRoot, ".env"))
	apiClient := os.Getenv("AMD")
	apiSecret := os.Getenv("AMS")

	// Debug: Print environment variables (without sensitive data)
	log.Printf("Environment variables:")
	log.Printf("  AMD (client_id): %s", apiClient)
	log.Printf("  AMS (client_secret): [HIDDEN]")

	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize services
	hotelService := services.NewHotelService(db)

	// Auto migrate models - updated to use new models
	err = db.AutoMigrate(&models.User{}, &models.Post{}, &models.HotelAPIItem{}, &models.HotelSearchData{}, &models.HotelRatingsData{}, &models.InvalidHotelSearchID{})
	if err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	ctx := context.Background()

	apiToken, err := oauth2_token(ctx, apiSecret, apiClient)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("API Token:", apiToken)

	// Get API endpoint URLs for different APIs
	hotelListURL := os.Getenv("AMADEUS_HOTEL_LIST_URL")
	if hotelListURL == "" {
		hotelListURL = "https://test.api.amadeus.com/v1/reference-data/locations/hotels/by-city"
	}

	hotelSearchURL := os.Getenv("AMADEUS_HOTEL_SEARCH_URL")
	if hotelSearchURL == "" {
		hotelSearchURL = "https://test.api.amadeus.com/v2/shopping/hotel-offers"
	}

	hotelRatingsURL := os.Getenv("AMADEUS_HOTEL_RATINGS_URL")
	if hotelRatingsURL == "" {
		hotelRatingsURL = "https://test.api.amadeus.com/v2/e-reputation/hotel-sentiments"
	}

	log.Printf("Using Hotel List URL: %s", hotelListURL)
	log.Printf("Using Hotel Search URL: %s", hotelSearchURL)
	log.Printf("Using Hotel Ratings URL: %s", hotelRatingsURL)

	// Step 1: Fetch hotel list using V1 API
	log.Println("=== Step 1: Fetching hotel list ===")
	hotelsCreated, err := fetchHotelsListPaginated(ctx, hotelService, hotelListURL, apiToken)
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
	// searchCreated, err := fetchHotelSearchData(ctx, hotelService, hotelSearchURL, apiToken, hotelIDs)
	// if err != nil {
	// 	log.Printf("Error fetching hotel search data: %v", err)
	// } else {
	// 	log.Printf("Successfully fetched search data for %d hotels", searchCreated)
	// }

	// Step 4: Fetch hotel ratings data using V2 API
	log.Println("=== Step 4: Fetching hotel ratings data ===")
	ratingsCreated, err := fetchHotelRatingsData(ctx, hotelService, hotelRatingsURL, apiToken, hotelIDs)
	if err != nil {
		log.Printf("Error fetching hotel ratings data: %v", err)
	} else {
		log.Printf("Successfully fetched ratings data for %d hotels", ratingsCreated)
	}

	// Summary
	log.Println("=== Summary ===")
	log.Printf("Hotels fetched: %d", hotelsCreated)
	//log.Printf("Search data fetched: %d", searchCreated)
	log.Printf("Ratings data fetched: %d", ratingsCreated)
}
