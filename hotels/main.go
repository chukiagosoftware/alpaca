package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
)

// City represents a city with IATA code
type City = models.AirportCity

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(currentFile))
	envPath := filepath.Join(projectRoot, ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Initialize database
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize storages
	hotelFetcher := newHotelFetcher(db)

	ctx := context.Background()

	// Get target cities from .env
	cities, err := getTargetCities(db)
	if err != nil {
		log.Fatalf("Failed to get target cities: %v", err)
	}

	totalAmadeusHotels := 0
	totalMultiSource := 0

	for _, city := range cities {
		log.Printf("Processing city: %s (%s)", city.Name, city.IATACode)

		// Fetch from Amadeus
		hotelIDs, err := fetchHotelsForCity(ctx, db, city.IATACode)
		log.Println("We are now getting custom list")

		if err != nil {
			log.Printf("Error fetching from Amadeus for %s: %v", city.IATACode, err)
		} else {
			totalAmadeusHotels += len(hotelIDs)
			log.Printf("Fetched %d hotels from Amadeus for %s", len(hotelIDs), city.IATACode)
			// Fetch detailed data for these hotels
			err = fetchDetailedDataForHotels(ctx, db, hotelIDs)
			if err != nil {
				log.Printf("Error fetching detailed data for %s: %v", city.IATACode, err)
			}
		}

		// Fetch from multi-source providers
		location := fmt.Sprintf("%s, %s", city.Name, city.Country)
		multiResults, err := hotelFetcher.fetchFromAllSources(ctx, location)
		if err != nil {
			log.Printf("Error in multi-source fetch for %s: %v", location, err)
		} else {
			for source, count := range multiResults {
				log.Printf("Fetched %d hotels from %s for %s", count, source, location)
				totalMultiSource += count
			}
		}

		// Rate limiting between cities
		time.Sleep(5 * time.Second)
	}

	log.Printf("Hotel fetching completed. Amadeus: %d, Multi-source: %d, Total: %d", totalAmadeusHotels, totalMultiSource, totalAmadeusHotels+totalMultiSource)

}

// getTargetCities retrieves cities from .env
func getTargetCities(db *orm.DB) ([]models.AirportCity, error) {
	citiesStr := os.Getenv("HOTEL_CITIES")
	if citiesStr == "" {
		citiesStr = "LON,AUS" // Default fallback
	}
	cityList := strings.Split(citiesStr, ",")
	for i, c := range cityList {
		cityList[i] = strings.TrimSpace(c)
	}

	var airportCities []models.AirportCity
	err := db.Where("iata_code IN ?", cityList).Find(&airportCities).Error
	return airportCities, err
}
