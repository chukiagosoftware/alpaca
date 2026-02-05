package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
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
func extractTopCities(airports []Airport) []City {
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
	type cityCount struct {
		city  City
		count int
	}

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

	// Extract top 500 cities
	var topCities []City
	for i, cc := range cityCounts {
		if i >= 500 {
			break
		}
		topCities = append(topCities, cc.city)
	}

	return topCities
}

// generateGoCode generates Go code for the TopCities slice
func generateGoCode(cities []City) string {
	var builder strings.Builder

	builder.WriteString("// TopCities is a slice of the top 500 cities in the world by air traffic.\n")
	builder.WriteString("// Generated from OpenFlights airports database.\n")
	builder.WriteString("var TopCities = []City{\n")

	for _, city := range cities {
		builder.WriteString(fmt.Sprintf("\t{Name: \"%s\", Country: \"%s\", IATACode: \"%s\"},\n",
			city.Name, city.Country, city.IATACode))
	}

	builder.WriteString("}\n")

	return builder.String()
}

// GenerateTopCities downloads airports data and generates Go code for top cities
func GenerateTopCities() error {
	log.Println("Starting to generate top cities data...")

	// Download airports data
	airports, err := downloadAirportsData()
	if err != nil {
		return fmt.Errorf("failed to download airports data: %w", err)
	}

	// Extract top cities
	cities := extractTopCities(airports)
	log.Printf("Extracted %d top cities", len(cities))

	// Generate Go code
	goCode := generateGoCode(cities)

	// Write to file
	filename := "generated_top_cities.go"
	err = os.WriteFile(filename, []byte(goCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	log.Printf("Generated Go code written to: %s", filename)
	log.Printf("Top 10 cities by airport count:")
	for i, city := range cities[:10] {
		log.Printf("%d. %s, %s (%s)", i+1, city.Name, city.Country, city.IATACode)
	}

	return nil
}
