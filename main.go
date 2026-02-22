package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chukiagosoftware/alpaca/database" // New: Import DB package
	"github.com/chukiagosoftware/alpaca/models"   // New: Import hotels package (for services and fetcher)
)

// main is the entry point - simplified to use packages
func main() {
	// Initialize database (now from package)
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize hotel service (now from hotels package)
	hotelService := hotels.NewhotelService(db)

	// Initialize multi-source fetcher (now from hotels package)
	multiSourceFetcher := hotels.NewMultiSourceFetcher(hotelService)

	// Example: Fetch from all sources for a location
	ctx := context.Background()
	location := "San Diego" // Or get from args/env
	results, err := multiSourceFetcher.FetchFromAllSources(ctx, location)
	if err != nil {
		log.Fatalf("Failed to fetch hotels: %v", err)
	}

	fmt.Printf("Fetched hotels: %+v\n", results)

	// Add any other top-level logic here (e.g., API server, CLI args)
	// For now, just exit after fetch
}

// Note: All the inline structs, methods, and providers have been moved to packages.
// - Database logic: database/database.go
// - Hotel service and providers: hotels/services.go, hotels/multiSource.go, etc.
// - If you have cities logic, import "cities" similarly.
