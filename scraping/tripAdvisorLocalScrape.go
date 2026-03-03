package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/gocolly/colly/v2"
)

// extractReviewsFromSavedHTML extracts reviews from a saved HTML file using Colly
func extractReviewsFromSavedHTML(filePath string) []map[string]string {
	var reviews []map[string]string

	// Create a custom http.Transport to handle the file:// protocol
	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))

	c := colly.NewCollector()
	c.WithTransport(t)

	c.OnHTML("div[data-test-target=\"HR_CC_CARD\"]", func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.DOM.Find("div._a a span span").Text())
		name := strings.TrimSpace(e.DOM.Find("div.ZRBpD.P._c div span a span").Text())
		text := strings.TrimSpace(e.DOM.Find("div._c div.fIrGe._T.bgMZj div span div span").Text())
		if title != "" || name != "" || text != "" {
			review := make(map[string]string, 3)
			review["Title"] = title
			review["Name"] = name
			review["Review"] = text
			reviews = append(reviews, review)
		}
	})

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("Error getting absolute path for %s: %v", filePath, err)
		return reviews
	}

	// Visit the local file
	fileURL := "file://" + filepath.ToSlash(absPath)

	// Parse the HTML string with Colly
	err = c.Visit(fileURL)
	if err != nil {
		log.Printf("Error visiting %s: %v", filePath, err)
	}

	return reviews
}

// processSavedHotelPages walks the saved HTML files and processes them
func processSavedHotelPages(db *orm.DB, ctx context.Context, baseDir string) (int, error) {
	var cnt int
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Parse path to get city, country
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) != 2 {
			log.Printf("Invalid path structure: %s", relPath)
			return nil
		}
		cityCountry := parts[0]
		cityCountryParts := strings.Split(cityCountry, ",")
		if len(cityCountryParts) != 2 {
			log.Printf("Invalid city,country: %s", cityCountry)
			return nil
		}
		city := cityCountryParts[0]
		country := cityCountryParts[1]

		// Parse filename: hotel_id_source_hotelName.html
		filename := parts[1]
		nameParts := strings.Split(strings.TrimSuffix(filename, ".html"), "_")
		if len(nameParts) < 3 {
			log.Printf("Invalid filename format: %s", filename)
			return nil
		}
		hotelID := nameParts[0]
		source := nameParts[1]
		hotelNameParts := nameParts[2:]
		hotelName := strings.Join(hotelNameParts, " ")

		// Extract reviews
		reviews := extractReviewsFromSavedHTML(path)

		// Create or update hotel
		hotelModel := &models.Hotel{
			HotelID:       hotelID,
			Source:        source,
			SourceHotelID: hotelID, // Assuming same
			Name:          hotelName,
			City:          city,
			Country:       country,
			LastUpdate:    time.Now().Format(time.RFC3339),
		}
		if err := db.CreateOrUpdateHotel(ctx, hotelModel); err != nil {
			log.Printf("Error saving hotel %s: %v", hotelName, err)
			return nil
		}

		// Save reviews
		for _, data := range reviews {
			hash := sha256.New()
			hash.Write([]byte(data["Review"]))
			review := &models.HotelReview{
				HotelID:        hotelID,
				Source:         source,
				SourceReviewID: hex.EncodeToString(hash.Sum(nil)),
				ReviewerName:   data["Name"],
				ReviewText:     data["Title"] + "\n" + data["Review"],
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := db.SaveReview(ctx, review); err != nil {
				log.Printf("Error saving review for %s: %s, error:%v", hotelName, data["Title"], err)
			} else {
				log.Printf("Saved review: %s\n%s\n", hotelName, data["Title"])
			}
		}
		cnt += len(reviews)
		log.Printf("Processed %s: %d reviews", hotelName, len(reviews))
		return nil
	})

	return cnt, err
}

func main() {
	// Initialize database
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	baseDir := "hotelReviewsSaved"

	if cnt, err := processSavedHotelPages(db, ctx, baseDir); err != nil {
		log.Fatalf("Error processing saved pages: %v", err)
	} else {
		log.Printf("Processing completed. Total: %d", cnt)
	}
}
