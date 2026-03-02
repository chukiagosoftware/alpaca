package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(currentFile))
	envPath := filepath.Join(projectRoot, ".env")

	if err := godotenv.Load(envPath); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Connect to DB
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Get HOTEL_CITIES
	hotelCitiesStr := os.Getenv("HOTEL_CITIES")
	if hotelCitiesStr == "" {
		log.Fatal("HOTEL_CITIES not set")
	}
	iataCodes := strings.Split(hotelCitiesStr, ",")

	// Build cities map
	citiesMap := make(map[string]string)
	for _, iata := range iataCodes {
		var airport models.AirportCity
		if err := db.Where("iata_code = ?", strings.TrimSpace(iata)).First(&airport).Error; err != nil {
			log.Printf("Error finding airport for %s: %v", iata, err)
			continue
		}
		cityName := airport.Name
		country := airport.Country

		// Search for location_id
		locationID := searchCityLocationID(cityName, country)
		if locationID != "" {
			citiesMap[cityName] = locationID
		}
		// Wait 2 seconds between city location searches to avoid rate limiting
		time.Sleep(2 * time.Second)
	}

	log.Printf("Cities map: %v", citiesMap)

	// Now, for each city, search hotels and save
	hotelCounter := 1
	for city, location := range citiesMap {
		// Parse location to get country
		parts := strings.Split(location, "-")
		if len(parts) < 2 {
			continue
		}
		cityCountry := parts[1]
		countryParts := strings.Split(cityCountry, "_")
		country := countryParts[len(countryParts)-1]

		hotels := searchHotels(location)
		// Wait 3 seconds after fetching hotels list for a city
		time.Sleep(3 * time.Second)

		for _, hotel := range hotels {
			// Save hotel
			hotelModel := &models.Hotel{
				HotelID:       fmt.Sprintf("tripadvisor_%d", hotelCounter),
				Source:        models.HotelSourceTripadvisor,
				SourceHotelID: hotel.ID,
				Name:          hotel.Name,
				City:          city,
				Country:       country,
				LastUpdate:    time.Now().Format(time.RFC3339),
			}
			if err := db.CreateOrUpdateHotel(ctx, hotelModel); err != nil {
				log.Printf("Error saving hotel: %v", err)
				continue
			}
			// Wait 1 second after saving hotel
			time.Sleep(1 * time.Second)

			// Get review
			reviewText := getHotelReview(hotel.URL)
			if reviewText != "" {
				review := &models.HotelReview{
					HotelID:    hotelModel.HotelID,
					Source:     models.SourceTripadvisor,
					ReviewText: reviewText,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				if err := db.SaveReview(ctx, review); err != nil {
					log.Printf("Error saving review: %v", err)
				}
			}
			// Wait 2 seconds after processing each hotel review
			time.Sleep(2 * time.Second)

			hotelCounter++
		}
		// Wait 5 seconds between processing different cities
		time.Sleep(5 * time.Second)
	}
}

type HotelInfo struct {
	ID   string
	Name string
	URL  string
}

func searchCityLocationID(city, country string) string {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
	// Set delay between requests to the same domain
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       2 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	var locationID string
	c.OnHTML("a[href*='/Hotels-g']", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		re := regexp.MustCompile(`/Hotels-g(\d+)-([^.]+)\.html`)
		matches := re.FindStringSubmatch(href)
		if len(matches) > 1 {
			locationID = "g" + matches[1] + "-" + matches[2]
		}
	})

	searchURL := fmt.Sprintf("https://www.tripadvisor.com/Search?q=%s+%s+hotels", url.QueryEscape(city), url.QueryEscape(country))
	c.Visit(searchURL)
	c.Wait()

	return locationID
}

func searchHotels(location string) []HotelInfo {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
	// Set delay between requests
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       3 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	var hotels []HotelInfo
	c.OnHTML("a[href*='/Hotel_Review-']", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		re := regexp.MustCompile(`/Hotel_Review-g\d+-d(\d+)-Reviews-([^.]+)\.html`)
		matches := re.FindStringSubmatch(href)
		if len(matches) > 2 {
			hotelID := "d" + matches[1]
			hotelName := strings.ReplaceAll(matches[2], "_", " ")
			hotels = append(hotels, HotelInfo{
				ID:   hotelID,
				Name: hotelName,
				URL:  "https://www.tripadvisor.com" + href,
			})
		}
	})

	hotelsURL := fmt.Sprintf("https://www.tripadvisor.com/Hotels-%s-Hotels.html", location)
	c.Visit(hotelsURL)
	c.Wait()

	return hotels
}

func getHotelReview(hotelURL string) string {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
	// Set delay for review scraping
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       2 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	var reviewText string
	c.OnHTML("#REVIEWS > div > div.aRZXW > div > div > div:nth-child(2) > div > div > div.mSOQy > div.FRFxD._u > div:nth-child(1) > div > div:nth-child(1) > div.FKRgy.f.e > div._c > div > div.fIrGe._T.bgMZj > div > span > div > span", func(e *colly.HTMLElement) {
		reviewText = e.Text
	})

	c.Visit(hotelURL)
	c.Wait()

	return reviewText
}
