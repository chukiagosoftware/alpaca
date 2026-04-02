package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env: %v", err)
	}

	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	var hotels []*models.Hotel
	db.Joins("LEFT JOIN cities ON hotels.city = cities.name AND hotels.country = cities.country").
		Where("cities.continent IN ?", []string{"us"}). //} "mexico", "canada", "europe"}).
		Find(&hotels)

	log.Printf("Found %d hotels for Yelp processing", len(hotels))

	for i, hotel := range hotels {
		if i < 6 {
			continue
			log.Printf("Skipping %s", hotel.Name)
		}
		log.Printf("[%d/%d] Processing %s", i+1, len(hotels), hotel.Name)

		safeName := sanitizeFilename(hotel.Name)
		subDir := filepath.Join("hotelReviewSaved", "yelp", fmt.Sprintf("%s,%s", hotel.City, hotel.Country))
		os.MkdirAll(subDir, 0755)

		searchPath := filepath.Join(subDir, fmt.Sprintf("search-%s-%s.html", safeName, hotel.HotelID))

		// === STAGE 1: Search and save search results page ===
		cmd := exec.Command("osascript", "scraping/yelp_applescript.scpt",
			hotel.Name, hotel.City, hotel.Country, hotel.HotelID, searchPath)
		cmd.Run()
		time.Sleep(6 * time.Second)

		// Extract first hotel URL from search page
		hotelURL, err := extractFirstYelpBizLink(searchPath)
		if err != nil || hotelURL == "" {
			log.Printf("No biz link found for %s", hotel.Name)
			continue
		}

		// === STAGE 2: Open hotel page and save it ===
		hotelPagePath := filepath.Join(subDir, fmt.Sprintf("reviews-%s-%s.html", safeName, hotel.HotelID))
		cmd2 := exec.Command("osascript", "scraping/yelp_applescript.scpt",
			hotel.Name, hotel.City, hotel.Country, hotel.HotelID, hotelPagePath, hotelURL)
		cmd2.Run()
		time.Sleep(6 * time.Second)

		log.Printf("Saved hotel page: %s", hotelPagePath)
		time.Sleep(2 * time.Second) // Be nice to Yelp
	}
}

func sanitizeFilename(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "_")
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func extractFirstYelpBizLink(htmlPath string) (string, error) {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`href="(/biz/[^"]+)"`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) > 1 {
		u, _ := url.Parse("https://www.yelp.com" + matches[1])
		return u.String(), nil
	}
	return "", fmt.Errorf("no /biz/ link found")
}
