package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
)

// hotelDataProvider defines the interface for hotel data providers
type hotelDataProvider interface {
	getProviderName() string
	fetchHotels(ctx context.Context, location string) ([]*models.Hotel, error)
	isEnabled() bool
}

// hotelFetcher coordinates fetching from multiple hotel sources
type hotelFetcher struct {
	db        *orm.DB
	providers []hotelDataProvider
}

// newHotelFetcher creates a new hotel fetcher
func newHotelFetcher(db *orm.DB) *hotelFetcher {
	return &hotelFetcher{
		db: db,
		providers: []hotelDataProvider{
			newGooglePlacesProvider(),
			//newYelpProvider(),
			newTripAdvisorProvider(),
		},
	}
}

// fetchFromAllSources fetches hotels from all enabled providers
func (f *hotelFetcher) fetchFromAllSources(ctx context.Context, location string) (map[string]int, error) {
	results := make(map[string]int)

	for _, provider := range f.providers {
		if !provider.isEnabled() {
			log.Printf("Provider %s is disabled (missing API key)", provider.getProviderName())
			results[provider.getProviderName()] = 0
			continue
		}

		log.Printf("Fetching hotels from %s for location: %s", provider.getProviderName(), location)

		hotels, err := provider.fetchHotels(ctx, location)
		if err != nil {
			log.Printf("Error fetching from %s: %v", provider.getProviderName(), err)
			results[provider.getProviderName()] = 0
			continue
		}

		// Save hotels to database
		savedCount := 0
		for _, hotel := range hotels {
			if err := f.db.CreateOrUpdateHotel(ctx, hotel); err != nil {
				log.Printf("Error saving hotel %s from %s: %v", hotel.Name, provider.getProviderName(), err)
				continue
			}
			savedCount++
		}

		results[provider.getProviderName()] = savedCount
		log.Printf("Saved %d hotels from %s", savedCount, provider.getProviderName())

		// Rate limiting between providers
		time.Sleep(2 * time.Second)
	}

	return results, nil
}

// parseStateFromAddress parses state from address string
func parseStateFromAddress(address string) string {
	parts := strings.Split(address, ",")
	if len(parts) >= 2 {
		stateZip := strings.TrimSpace(parts[len(parts)-2])
		stateParts := strings.Fields(stateZip)
		if len(stateParts) > 0 {
			return stateParts[0]
		}
	}
	return ""
}
