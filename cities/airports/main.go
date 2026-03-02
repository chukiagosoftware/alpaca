package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
	"gorm.io/gorm/clause"
)

// Airport represents an airport from the OpenFlights dataset
type Airport struct {
	ID       string
	Name     string
	City     string
	Country  string
	IATA     string
	ICAO     string
	Lat      string
	Lon      string
	Altitude string
	Timezone string
	DST      string
	TzDB     string
	Type     string
	Source   string
}

// City represents a city with IATA code for the constants
type City struct {
	Name     string
	Country  string
	IATACode string
}

type cityCount struct {
	city  City
	count int
}

// downloadAirportsData downloads the OpenFlights airports dataset
func downloadAirportsData() ([]Airport, error) {
	url := "https://raw.githubusercontent.com/jpatokal/openflights/master/data/airports.dat"

	log.Printf("Downloading airports data from: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download airports data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download airports data: status %d", resp.StatusCode)
	}

	var airports []Airport
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		// OpenFlights uses a custom format with commas but fields can contain commas
		// We need to parse it carefully
		fields := parseOpenFlightsLine(line)

		if len(fields) >= 14 {
			airport := Airport{
				ID:       fields[0],
				Name:     fields[1],
				City:     fields[2],
				Country:  fields[3],
				IATA:     fields[4],
				ICAO:     fields[5],
				Lat:      fields[6],
				Lon:      fields[7],
				Altitude: fields[8],
				Timezone: fields[9],
				DST:      fields[10],
				TzDB:     fields[11],
				Type:     fields[12],
				Source:   fields[13],
			}
			airports = append(airports, airport)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading airports data: %w", err)
	}

	log.Printf("Downloaded %d airports", len(airports))
	return airports, nil
}

// parseOpenFlightsLine parses a line from the OpenFlights dataset
// The format uses commas as separators but fields can contain commas
func parseOpenFlightsLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		char := line[i]

		if char == '"' {
			inQuotes = !inQuotes
		} else if char == ',' && !inQuotes {
			fields = append(fields, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(char)
		}
	}

	// Add the last field
	fields = append(fields, strings.TrimSpace(current.String()))

	return fields
}

// extractTopCities processes airports data and returns the top cities
func extractTopCities(airports []Airport) []cityCount {
	// Create a map to deduplicate cities and count airports per city
	cityMap := make(map[string]int)
	cityInfo := make(map[string]City)

	for _, airport := range airports {
		// Skip airports without IATA codes or with invalid codes
		if airport.IATA == "" || airport.IATA == "\\N" || len(airport.IATA) != 3 {
			continue
		}

		// Skip non-passenger airports
		if airport.Type != "airport" {
			continue
		}

		// Create a unique key for city+country
		key := fmt.Sprintf("%s,%s", airport.City, airport.Country)

		// Count airports per city
		cityMap[key]++

		// Store city info (use the first occurrence)
		if _, exists := cityInfo[key]; !exists {
			cityInfo[key] = City{
				Name:     airport.City,
				Country:  airport.Country,
				IATACode: airport.IATA,
			}
		}
	}

	// Convert to slice and sort by airport count (descending)

	var cityCounts []cityCount
	for key, count := range cityMap {
		city := cityInfo[key]
		cityCounts = append(cityCounts, cityCount{city: city, count: count})
	}

	// Sort by airport count (descending), then by city name
	sort.Slice(cityCounts, func(i, j int) bool {
		if cityCounts[i].count != cityCounts[j].count {
			return cityCounts[i].count > cityCounts[j].count
		}
		return cityCounts[i].city.Name < cityCounts[j].city.Name
	})

	// Extract top num cities
	var topCityCounts []cityCount
	for i, cc := range cityCounts {
		if i >= 1000 {
			break
		}

		topCityCounts = append(topCityCounts, cc)
	}
	return topCityCounts
}

// generateTopCities downloads airports data and generates Go code for top cities
func generateTopCities(db *orm.DB) error {
	log.Println("Starting to generate top cities data...")

	// Download airports data
	airports, err := downloadAirportsData()
	if err != nil {
		return fmt.Errorf("failed to download airports data: %w", err)
	}

	// Extract top cities
	topCities := extractTopCities(airports)
	log.Printf("Extracted %d top cities", len(topCities))

	// Insert into database using GORM
	for _, cc := range topCities {
		city := models.AirportCity{
			Name:         cc.city.Name,
			Country:      cc.city.Country,
			IATACode:     cc.city.IATACode,
			AirportCount: cc.count,
		}

		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "iata_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "country", "airport_count"}),
		}).Create(&city).Error; err != nil {
			return fmt.Errorf("failed to insert city %s: %w", cc.city.Name, err)
		}
	}

	log.Printf("Inserted %d cities into database", len(topCities))

	log.Printf("Top 10 cities by airport count:")
	for i, cc := range topCities[:min(10, len(topCities))] {
		log.Printf("%d. %s, %s (%s) - %d airports", i+1, cc.city.Name, cc.city.Country, cc.city.IATACode, cc.count)
	}

	return nil
}

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	// Adjust path as needed; assuming cmd/airportcities/main.go
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
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

	// Generate top cities data
	if err := generateTopCities(db); err != nil {
		log.Fatalf("Failed to generate top cities: %v", err)
	}

	log.Println("Airport cities population completed successfully.")
}
