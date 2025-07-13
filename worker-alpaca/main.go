package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edamsoft-sre/alpaca/database"
	"github.com/edamsoft-sre/alpaca/models"
	"github.com/edamsoft-sre/alpaca/services"
	"github.com/joho/godotenv"
)

// fetchHotelsListPaginated fetches all hotels with proper pagination handling
func fetchHotelsListPaginated(ctx context.Context, hotelService *services.HotelService, baseURL, apiToken string) (int, error) {
	hotels_created := 0
	searchField := "cityCode"
	searchValue := "AUS"
	page := 0
	limit := 50 // Amadeus default limit

	for {
		data := url.Values{}
		data.Set(searchField, searchValue)
		data.Set("page[limit]", strconv.Itoa(limit))
		data.Set("page[offset]", strconv.Itoa(page*limit))

		requestURL := baseURL + "?" + data.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
		if err != nil {
			return hotels_created, fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiToken)

		log.Printf("Fetching page %d for cityCode: %s", page+1, searchValue)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return hotels_created, fmt.Errorf("error fetching %s: %w", requestURL, err)
		}
		defer resp.Body.Close()

		log.Printf("Status: %s", resp.Status)

		var apiResp models.HotelsListResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return hotels_created, fmt.Errorf("decode error: %w", err)
		}

		log.Printf("Page %d: Found %d hotels (Total: %d)", page+1, len(apiResp.Data), apiResp.Meta.Count)

		if len(apiResp.Data) == 0 {
			log.Println("No more hotels found, stopping pagination")
			break
		}

		for _, hotel := range apiResp.Data {
			log.Printf("Processing: %s", hotel.Name)
			err := hotelService.Create(ctx, &hotel)
			if err != nil {
				log.Printf("Error saving hotel %s: %v", hotel.Name, err)
			} else {
				hotels_created++
			}
		}

		// Check if we've fetched all hotels
		if len(apiResp.Data) < limit || hotels_created >= apiResp.Meta.Count {
			log.Printf("Reached end of results. Total fetched: %d", hotels_created)
			break
		}

		page++

		// Rate limiting - be nice to the API
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
			requestURL := baseURL + "/v2/shopping/hotel-offers?" + data.Encode()

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
				err := hotelService.CreateSearchData(ctx, &hotelData)
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
	semaphore := make(chan struct{}, 5) // Limit concurrent requests
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
			requestURL := baseURL + "/v2/e-reputation/hotel-sentiments?hotelIds=" + hotelID

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
				err := hotelService.CreateRatingsData(ctx, &ratingData)
				if err != nil {
					log.Printf("Error saving ratings data for hotel %s: %v", hotelID, err)
				} else {
					log.Printf("Fetched and saved ratings data for hotel: %s (Rating: %d)", hotelID, ratingData.OverallRating)

					mu.Lock()
					ratingsCreated++
					mu.Unlock()
				}
			}

			// Rate limiting
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
	_ = godotenv.Load("../.env")
	apiClient := os.Getenv("AMD")
	apiSecret := os.Getenv("AMS")
	baseURL := os.Getenv("HOTEL_API_URL") // e.g. "https://test.api.amadeus.com"
	byCityUrl := os.Getenv("BY_CITY_URL")

	url := baseURL + byCityUrl

	ctx := context.Background()
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer func() {
		sqlDB, err := db.DB()
		if err != nil {
			log.Printf("Error getting underlying DB: %v", err)
			return
		}
		sqlDB.Close()
	}()

	// Auto migrate models - updated to use new models
	err = db.AutoMigrate(&models.User{}, &models.Post{}, &models.HotelAPIItem{}, &models.HotelSearchData{}, &models.HotelRatingsData{})
	if err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	hotelService := services.NewHotelService(db)

	apiToken, err := oauth2_token(ctx, apiSecret, apiClient)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("API Token:", apiToken)

	// Step 1: Fetch basic hotel list with pagination
	log.Println("=== Step 1: Fetching hotel list ===")
	hotelsCount, err := fetchHotelsListPaginated(ctx, hotelService, url, apiToken)
	if err != nil {
		log.Fatal("Error fetching hotels:", err)
	}

	// Step 2: Get hotel IDs for detailed data fetching
	log.Println("=== Step 2: Getting hotel IDs ===")
	hotelIDs, err := getHotelIDs(ctx, hotelService)
	if err != nil {
		log.Fatal("Error getting hotel IDs:", err)
	}

	// Step 3: Fetch detailed search data (concurrent)
	log.Println("=== Step 3: Fetching hotel search data ===")
	searchCount, err := fetchHotelSearchData(ctx, hotelService, baseURL, apiToken, hotelIDs)
	if err != nil {
		log.Printf("Error fetching search data: %v", err)
	}

	// Step 4: Fetch ratings data (concurrent)
	log.Println("=== Step 4: Fetching hotel ratings data ===")
	ratingsCount, err := fetchHotelRatingsData(ctx, hotelService, baseURL, apiToken, hotelIDs)
	if err != nil {
		log.Printf("Error fetching ratings data: %v", err)
	}

	log.Printf("=== Summary ===")
	log.Printf("Hotels fetched: %d", hotelsCount)
	log.Printf("Search data fetched: %d", searchCount)
	log.Printf("Ratings data fetched: %d", ratingsCount)
}
