package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Country parsed from countries.dat (Name,ISO2,AltCode - comma-separated, quoted)
type Country struct {
	Name  string
	ISO2  string
	Codes string // Single if ISO2==AltCode, else "ISO2,AltCode"
}

// downloadAirportsData downloads the OpenFlights airports dataset (reused as is)
func downloadAirportsData() ([]models.Airport, error) {

	airportsList, err := os.Open("internal/airports/cities/airports.dat")
	if err != nil {
		return nil, fmt.Errorf("failed to open airports.dat: %w", err)
	}

	var airports []models.Airport
	scanner := bufio.NewScanner(airportsList)

	for scanner.Scan() {
		line := scanner.Text()
		// OpenFlights uses a custom format with commas but fields can contain commas
		// We need to parse it carefully
		fields := parseOpenFlightsLine(line)

		if len(fields) >= 14 {
			airport := models.Airport{
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

// downloadCitiesCountries fetches countries.dat, processes airports with city-focused selection, saves to airport_cities
// Logic: Unique cities per country; add all if <=25, top 25 (by AirportCount=#airports tie-breaker) if more
// One row per selected city: Representative IATA (first), aggregated AirportCount (# airports), CountryCodes
func downloadCitiesCountries(db *gorm.DB) error {
	maxPerCountryStr := os.Getenv("MAX_CITIES_PER_COUNTRY")
	maxPerCountry := 25
	if maxPerCountryStr != "" {
		if val, err := strconv.Atoi(maxPerCountryStr); err == nil {
			maxPerCountry = val
		}
	}
	fmt.Printf("maxPerCountry = %s\n", maxPerCountry)

	countriesFile, err := os.Open("internal/airports/cities/countries.dat")
	if err != nil {
		return fmt.Errorf("failed to open countries.dat: %w", err)
	}

	countries := make(map[string]Country)
	scanner := bufio.NewScanner(countriesFile)
	for scanner.Scan() {
		line := scanner.Text()
		fields := parseCSVLine(line)
		if len(fields) < 3 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		iso2 := strings.TrimSpace(fields[1])
		altCode := strings.TrimSpace(fields[2])

		if iso2 == "KP" {
			continue
		}

		var codes string
		if iso2 == altCode {
			codes = iso2
		} else {
			codes = fmt.Sprintf("%s,%s", iso2, altCode)
		}

		countries[iso2] = Country{Name: name, ISO2: iso2, Codes: codes}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("countries scan error: %w", err)
	}
	log.Printf("Loaded %d countries (sample: %s -> %s)", len(countries), "AG", countries["AG"].Codes)

	// Step 2: Fetch airports
	airports, err := downloadAirportsData()
	if err != nil {
		return fmt.Errorf("failed to download airports: %w", err)
	}

	// Step 3: Group by country ISO2 -> map[cityLower] = []Airport (all for that city)
	countryCityMap := make(map[string]map[string][]models.Airport) // ISO2 -> cityLower -> []Airport
	skipped := 0
	for _, airport := range airports {
		var iso2 string
		for cISO, c := range countries {
			if strings.EqualFold(airport.Country, c.Name) {
				iso2 = cISO
				break
			} else if strings.Contains(strings.ToLower(airport.Country), strings.ToLower(c.Name)) {
				iso2 = cISO // Fuzzy
				break
			}
		}
		if iso2 == "" {
			skipped++
			continue
		}

		cityLower := strings.ToLower(airport.City)
		if countryCityMap[iso2] == nil {
			countryCityMap[iso2] = make(map[string][]models.Airport)
		}
		countryCityMap[iso2][cityLower] = append(countryCityMap[iso2][cityLower], airport)
	}
	log.Printf("Grouped %d airports into %d countries (%d skipped; unique cities per country below)", len(airports)-skipped, len(countryCityMap), skipped)

	// Step 4: Per country, select unique cities, prepare one row per
	var selectedCities []models.AirportCity
	totalUniqueCities := 0
	for iso2, cityMap := range countryCityMap {
		uniqueCities := len(cityMap)
		totalUniqueCities += uniqueCities
		countryName := countries[iso2].Name

		log.Printf("Country %s (%s): %d unique cities", iso2, countryName, uniqueCities)

		if uniqueCities <= maxPerCountry {
			// Add all cities (one row each)
			for _, apts := range cityMap {
				cityName := apts[0].City  // Original case
				repIATA := apts[0].IATA   // First as representative
				airportCount := len(apts) // # airports for this city

				selectedCities = append(selectedCities, models.AirportCity{
					IATACode:     repIATA,
					Name:         cityName,
					Country:      countryName,
					AirportCount: airportCount,
					CountryCodes: countries[iso2].Codes,
				})
			}
			log.Printf("  - Added all %d cities (each with aggregated AirportCount)", uniqueCities)
			continue
		}

		// >25: Tie-break sort by AirportCount (# airports DESC), then name ASC
		var citySummaries []struct {
			Name         string
			AirportCount int // len(apts)
			RepIATA      string
		}
		for _, apts := range cityMap {
			cityName := apts[0].City
			repIATA := apts[0].IATA
			airportCount := len(apts)

			citySummaries = append(citySummaries, struct {
				Name         string
				AirportCount int
				RepIATA      string
			}{Name: cityName, AirportCount: airportCount, RepIATA: repIATA})
		}

		// Sort
		sort.Slice(citySummaries, func(i, j int) bool {
			if citySummaries[i].AirportCount != citySummaries[j].AirportCount {
				return citySummaries[i].AirportCount > citySummaries[j].AirportCount
			}
			return citySummaries[i].Name < citySummaries[j].Name
		})

		// Select top 25
		numSelect := maxPerCountry // 25
		dropped := uniqueCities - numSelect
		log.Printf("  - Selected top %d cities (dropped %d; tie-breaker: # airports per city)", numSelect, dropped)
		for i := 0; i < numSelect; i++ {
			summary := citySummaries[i]
			selectedCities = append(selectedCities, models.AirportCity{
				IATACode:     summary.RepIATA,
				Name:         summary.Name,
				Country:      countryName,
				AirportCount: summary.AirportCount,
				CountryCodes: countries[iso2].Codes,
			})
		}
	}
	log.Printf("Overall: %d total unique cities processed, %d selected cities to save (one row each)", totalUniqueCities, len(selectedCities))

	// Step 5: Save with GORM (one per city)
	for _, city := range selectedCities {
		err = db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "iata_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "country", "airport_count", "country_codes"}),
		}).Create(&city).Error
		if err != nil {
			log.Printf("Failed insert for %s (%s): %v", city.Name, city.IATACode, err)
		}
	}

	return nil
}

// parseOpenFlightsLine parses a line from the OpenFlights dataset (reused as is)
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

// parseCSVLine for countries.dat (comma-separated with quotes)
func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	var inQuote bool
	runes := []rune(line)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == '"' {
			inQuote = !inQuote
			i++
			continue
		}
		if r == ',' && !inQuote {
			fields = append(fields, strings.TrimSpace(current.String()))
			current.Reset()
			i++
			continue
		}
		current.WriteRune(r)
		i++
	}
	fields = append(fields, strings.TrimSpace(current.String()))
	return fields
}

// generateTopCities now calls downloadCitiesCountries (simplified)
func generateTopCities(db *gorm.DB) error {
	log.Println("Starting to generate top cities data...")

	if err := downloadCitiesCountries(db); err != nil {
		return fmt.Errorf("failed to download cities countries: %w", err)
	}

	// Total unique cities saved
	var total int64
	db.Model(&models.AirportCity{}).Count(&total)
	log.Printf("Total saved unique cities: %d (one row each)", total)

	// Top 10 by AirportCount DESC
	var top10 []models.AirportCity
	err := db.Order("airport_count DESC, name ASC").Limit(10).Find(&top10).Error
	if err != nil {
		return fmt.Errorf("failed to query top cities: %w", err)
	}

	log.Printf("Top 10 cities by # airports (tie-breaker):")
	for i, city := range top10 {
		log.Printf("%d. %s, %s (Rep IATA: %s, # Airports: %d, Codes: %s)", i+1, city.Name, city.Country, city.IATACode, city.AirportCount, city.CountryCodes)
	}

	log.Println("Airport cities population completed successfully.")
	return nil
}

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(currentFile))))
	envPath := filepath.Join(projectRoot, ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Initialize database (handles migration)
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Generate top cities data
	if err := generateTopCities(db.DB); err != nil {
		log.Fatalf("Failed to generate top cities: %v", err)
	}

	log.Println("Airport cities population completed successfully.")
}
