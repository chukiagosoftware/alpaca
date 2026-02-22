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

	"github.com/chukiagosoftware/alpaca/database"
	"github.com/joho/godotenv"
)

// City represents a city with IATA code
type City struct {
	Name     string
	Country  string
	IATACode string
}

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(currentFile))
	envPath := filepath.Join(projectRoot, ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize storages
	hotelStorage := newHotelStorage(db)
	hotelFetcher := newHotelFetcher(hotelStorage)

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
		amadeusHotels, err := fetchHotelsForCity(ctx, hotelStorage, city.IATACode)
		if err != nil {
			log.Printf("Error fetching from Amadeus for %s: %v", city.IATACode, err)
		} else {
			totalAmadeusHotels += amadeusHotels
			log.Printf("Fetched %d hotels from Amadeus for %s", amadeusHotels, city.IATACode)
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
func getTargetCities(db *database.DB) ([]City, error) {
	citiesStr := os.Getenv("HOTEL_CITIES")
	if citiesStr == "" {
		citiesStr = "LON,AUS" // Default fallback
	}
	cityList := strings.Split(citiesStr, ",")
	for i, c := range cityList {
		cityList[i] = strings.TrimSpace(c)
	}

	// Build dynamic IN clause with placeholders
	placeholders := strings.Repeat("?,", len(cityList))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
	query := "SELECT name, country, iata_code FROM airport_cities WHERE iata_code IN (" + placeholders + ")"

	args := make([]interface{}, len(cityList))
	for i, c := range cityList {
		args[i] = c
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []City
	for rows.Next() {
		var city City
		if err := rows.Scan(&city.Name, &city.Country, &city.IATACode); err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}
	return cities, rows.Err()
}
