package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/models"
)

type googlePlacesResponse struct {
	Places []struct {
		ID          string `json:"id"`
		DisplayName struct {
			Text string `json:"text"`
		} `json:"displayName"`
		FormattedAddress string `json:"formattedAddress"`
		Location         struct {
			LatLng struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"latLng"`
		} `json:"location"`
		Rating          float64 `json:"rating"`
		UserRatingCount int     `json:"userRatingCount"`
	} `json:"places"`
	NextPageToken string `json:"nextPageToken"`
}

type googlePlacesRequest struct {
	TextQuery      string `json:"textQuery"`
	MaxResultCount int    `json:"maxResultCount"`
	PageToken      string `json:"pageToken,omitempty"`
}

type googlePlacesProvider struct {
	apiKey string
	client *http.Client
}

func newGooglePlacesProvider() *googlePlacesProvider {
	return &googlePlacesProvider{
		apiKey: os.Getenv("GOOGLE_PLACES_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *googlePlacesProvider) getProviderName() string {
	return models.HotelSourceGoogle
}

func (p *googlePlacesProvider) isEnabled() bool {
	return p.apiKey != ""
}

func (p *googlePlacesProvider) fetchHotels(ctx context.Context, location string) ([]*models.Hotel, error) {
	baseURL := "https://places.googleapis.com/v1/places:searchText"

	searchStringsEnv := os.Getenv("GOOGLE_PLACES_SEARCH_STRINGS")
	if searchStringsEnv == "" {
		searchStringsEnv = "quiet hotels near,best hotels near,cheap hotels near,quality hotels in" // Default
	}
	searchStrings := strings.Split(searchStringsEnv, ",")
	for i := range searchStrings {
		searchStrings[i] = strings.TrimSpace(searchStrings[i])
	}

	var allHotels []*models.Hotel
	hotelMap := make(map[string]*models.Hotel) // To dedupe by hotel_id

	for _, searchPrefix := range searchStrings {
		pageToken := ""
		maxPages := 3

		for page := 0; page < maxPages; page++ {
			requestBody := googlePlacesRequest{
				TextQuery:      fmt.Sprintf("%s %s", searchPrefix, location),
				MaxResultCount: 20,
				PageToken:      pageToken,
			}

			jsonData, err := json.Marshal(requestBody)
			if err != nil {
				return nil, fmt.Errorf("error marshaling request: %w", err)
			}

			req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(jsonData))
			if err != nil {
				return nil, fmt.Errorf("error creating request: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Goog-Api-Key", p.apiKey)
			req.Header.Set("X-Goog-FieldMask", "places.id,places.displayName,places.formattedAddress,places.location,places.rating,places.userRatingCount")

			resp, err := p.client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("error making request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
			}

			var placesResp googlePlacesResponse
			if err := json.NewDecoder(resp.Body).Decode(&placesResp); err != nil {
				return nil, fmt.Errorf("error decoding response: %w", err)
			}

			for _, place := range placesResp.Places {
				lat := place.Location.LatLng.Latitude
				lng := place.Location.LatLng.Longitude
				rating := place.Rating
				// Debug logging for lat/lng
				log.Printf("Hotel %s (%s): lat=%f, lng=%f", place.ID, place.DisplayName.Text, lat, lng)

				city, country, state := parseAddressComponents(place.FormattedAddress)

				hotelID := fmt.Sprintf("google_%s", place.ID)
				if _, exists := hotelMap[hotelID]; !exists {
					hotel := &models.Hotel{
						HotelID:       hotelID,
						Source:        models.HotelSourceGoogle,
						SourceHotelID: place.ID,
						Name:          place.DisplayName.Text,
						City:          city,
						Country:       country,
						StreetAddress: place.FormattedAddress,
						StateCode:     state,
						Latitude:      &lat,
						Longitude:     &lng,
						GoogleRating:  &rating,
						LastUpdate:    time.Now().Format(time.RFC3339),
					}
					hotelMap[hotelID] = hotel
					allHotels = append(allHotels, hotel)
				}
			}

			log.Printf("Fetched page %d for '%s %s': %d hotels (total unique: %d)", page+1, searchPrefix, location, len(placesResp.Places), len(allHotels))

			if placesResp.NextPageToken == "" {
				break
			}
			pageToken = placesResp.NextPageToken
			time.Sleep(2 * time.Second)
		}

		// Rate limiting between search strings
		time.Sleep(1 * time.Second)
	}

	return allHotels, nil
}

// ... existing code ...

// parseAddressComponents parses city, country, and state from formatted address
func parseAddressComponents(address string) (city, country, state string) {
	parts := strings.Split(address, ",")
	partsLen := len(parts)
	if partsLen == 0 {
		return
	}

	// Trim spaces from parts
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// Country is typically the last part
	country = parts[partsLen-1]

	// Handle different formats based on country
	if country == "USA" || country == "United States" {
		// US format: street, city, state zip, country
		if partsLen >= 3 {
			stateZip := parts[partsLen-2]
			fields := strings.Fields(stateZip)
			if len(fields) >= 2 {
				state = fields[0] // State code (e.g., CO)
				city = parts[partsLen-3]
			}
		}
	} else if country == "Australia" {
		// Australia format: street, city state postal, country
		if partsLen >= 3 {
			cityState := parts[partsLen-2]
			fields := strings.Fields(cityState)
			if len(fields) >= 2 {
				city = strings.Join(fields[:len(fields)-2], " ") // City name
				state = fields[len(fields)-2]                    // State (e.g., NSW)
			}
		}
	} else if country == "Brazil" {
		// Brazil format: street, neighborhood, city - state, postal, country
		if partsLen >= 4 {
			cityState := parts[partsLen-3]
			if strings.Contains(cityState, " - ") {
				cityStateParts := strings.Split(cityState, " - ")
				if len(cityStateParts) >= 2 {
					city = cityStateParts[0]
					state = cityStateParts[1]
				}
			}
		}
	} else {
		// General fallback for other countries: second last is city + postal
		if partsLen >= 2 {
			cityPostal := parts[partsLen-2]
			city = cleanCityFromPostal(cityPostal)
		}
	}

	return
}

// cleanCityFromPostal removes postal code from city string for general cases
func cleanCityFromPostal(cityPostal string) string {
	fields := strings.Fields(cityPostal)
	if len(fields) == 0 {
		return cityPostal
	}

	// Heuristic: If last field looks like postal code, remove it
	last := fields[len(fields)-1]
	if isLikelyPostalCode(last) {
		return strings.Join(fields[:len(fields)-1], " ")
	}

	// If first field looks like postal, remove it
	if isLikelyPostalCode(fields[0]) {
		return strings.Join(fields[1:], " ")
	}

	// Otherwise, return as is
	return cityPostal
}

// isLikelyPostalCode checks if a string looks like a postal code
func isLikelyPostalCode(s string) bool {
	// Simple checks: all digits (US), or alphanumeric with spaces (UK), etc.
	if len(s) < 3 || len(s) > 10 {
		return false
	}
	hasDigit := false
	hasLetter := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			hasDigit = true
		} else if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasLetter = true
		} else if r != ' ' {
			return false // Non-alphanumeric
		}
	}
	return hasDigit || hasLetter // Must have at least one digit or letter
}
