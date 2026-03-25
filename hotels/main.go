package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
)

// We now use City from topCities instead
// type City = models.AirportCity

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
		log.Printf("Processing city: %s (%s)", city.Name, city.Country)

		// Fetch from Amadeus
		//hotelIDs, err := fetchHotelsForCity(ctx, db, city.IATACode)
		//log.Println("We are now getting custom list")
		//
		//if err != nil {
		//	log.Printf("Error fetching from Amadeus for %s: %v", city.IATACode, err)
		//} else {
		//	totalAmadeusHotels += len(hotelIDs)
		//	log.Printf("Fetched %d hotels from Amadeus for %s", len(hotelIDs), city.IATACode)
		//	// Fetch detailed data for these hotels
		//	err = fetchDetailedDataForHotels(ctx, db, hotelIDs)
		//	if err != nil {
		//		log.Printf("Error fetching detailed data for %s: %v", city.IATACode, err)
		//	}
		//}

		// Fetch from enabled API providers
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

func getTargetCities(db *orm.DB) ([]models.City, error) {

	var cities []models.City
	err := db.Table("cities"). // Start from cities table
					Joins("LEFT JOIN hotels ON cities.name = hotels.city AND cities.country = hotels.country").
					Where("hotels.id IS NULL"). // Only cities with no matching hotels
					Find(&cities).Error
	if err != nil {
		return nil, err
	}
	fmt.Printf("Loaded %d cities\n", len(cities))

	return cities, err
}
