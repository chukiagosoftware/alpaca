package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"strconv"

	"github.com/edamsoft-sre/alpaca/models"
	"github.com/joho/godotenv"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func initDB() *gorm.DB {
	var err error
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Migrate the schema for Hotels for now
	db.AutoMigrate(&models.HotelAPIItem{})
	return db
}

func fetchHotelsListPaginated(ctx context.Context, db *gorm.DB, baseURL, apiToken string) (int, error) {

	hotels_created := 0
	searchField := "cityCode"
	searchValue := "AUS"


	for {

		data := url.Values{}
		data.Set(searchField, searchValue)
		baseURL = baseURL + "?" + data.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)

		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer " + apiToken)
		
		log.Printf("Get cityCode: %s with url: %s", searchValue, req.URL.String())

		resp, err := http.DefaultClient.Do(req)
		log.Printf("Status:%s", resp.Status)
		if err != nil {
			return 0, fmt.Errorf("error fetching %s: %w", baseURL, err)
		}
		defer resp.Body.Close()

		var apiResp models.HotelsListResponse
	

		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return 0, fmt.Errorf("decode error: %w", err)
		}

		log.Println(apiResp.Meta)
		

		if apiResp.Meta.Count == 0 {
			log.Println("No hotels found in the response")
			return 0, fmt.Errorf("no hotels found in the response")

		}

		for _, hotel := range apiResp.Data {
			log.Println(hotel.Name)
			db.Create(&hotel)
			hotels_created++
			
		}
		db.Commit()

		// Detect end of stream
		if apiResp.Meta.Next != "" {
			url := apiResp.Meta.Next // Update URL for next page
			log.Printf("Fetched %d hotels, moving to next page: %s", len(apiResp.Data), url)
			continue
		}

		break
	}
	return hotels_created, nil
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
	baseURL := os.Getenv("HOTEL_API_URL") // e.g. "https://api.example.com/hotels"
	byCityUrl := os.Getenv("BY_CITY_URL")

	url := baseURL + byCityUrl

	ctx := context.Background()
	db := initDB()
	if db == nil {
		log.Fatal("Failed to initialize database")
	}

	err := db.AutoMigrate(&models.HotelApiAmadeus{
		}, &models.RatingsAmadeus{
	})
	if err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}
	apiToken, err := oauth2_token(ctx, apiSecret, apiClient)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("API Token:", apiToken)

	if cnt, err := fetchHotelsListPaginated(ctx, db, url, apiToken); err != nil {
		log.Fatal(cnt, err)
	}

	