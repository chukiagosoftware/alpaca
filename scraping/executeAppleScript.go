package main

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
)

func main() {
	city := "MIAMI"
	country := "US"

	// Initialize database
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	var hotels []models.Hotel
	err = db.Select("city", "country", "hotel_id", "name").Where("city = ? AND country = ?", city, country).Find(&hotels).Error
	if err != nil {
		log.Fatalf("Failed to query hotels: %v", err)
	}

	// Path to the AppleScript file (relative to project root)
	scriptPath := "./scraping/tripadvisor_applescript.scpt"

	for _, hotel := range hotels {
		fmt.Printf("Processing hotel: %s (%s)\n", hotel.Name, hotel.HotelID)

		// Run osascript with the script and arguments: hotelName, city, country, hotelID
		cmd := exec.Command("osascript", scriptPath, hotel.Name, hotel.City, hotel.Country, hotel.HotelID)
		err := cmd.Run()
		time.Sleep(2 * time.Second)
		if err != nil {
			log.Printf("Error running AppleScript for hotel %s: %v", hotel.Name, err)
			continue // Continue with next hotel
		}

		fmt.Printf("Successfully processed hotel: %s\n", hotel.Name)

		// Pause for half a second before the next hotel
		time.Sleep(4 * time.Second)
	}

	fmt.Println("All hotels processed.")
}
