package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func extractYelpReviewsFromSavedHTML(filePath string) []map[string]string {
	var reviews []map[string]string

	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))

	c := colly.NewCollector()
	c.WithTransport(t)

	c.OnHTML("#reviews section div.y-css-mhg9c5 ul li", func(e *colly.HTMLElement) {
		name := strings.TrimSpace(e.DOM.Find("div.y-css-9vtc3g div.y-css-8x4us div").Text())
		text := strings.TrimSpace(e.DOM.Find("div.y-css-mhg9c5 div:nth-child(3) p").Text())
		if text == "" {
			text = strings.TrimSpace(e.ChildText("p"))
		}

		ratingStr := e.ChildAttr("div[role=\"img\"][aria-label*=\"star rating\"]", "aria-label")
		rating := 0.0
		if ratingStr != "" {
			re := regexp.MustCompile(`(\d+) star`)
			if match := re.FindStringSubmatch(ratingStr); len(match) > 1 {
				if val, err := strconv.ParseFloat(match[1], 64); err == nil {
					rating = val
				}
			}
		}

		if text != "" {
			review := make(map[string]string, 4)
			review["Name"] = name
			review["Review"] = text
			review["Rating"] = strconv.FormatFloat(rating, 'f', 1, 64)
			reviews = append(reviews, review)
		}
	})

	absPath, _ := filepath.Abs(filePath)
	fileURL := "file://" + filepath.ToSlash(absPath)
	c.Visit(fileURL)

	return reviews
}

func processYelpSavedHotelPages(db *gorm.DB, ctx context.Context, baseDir string) (int, error) {
	var totalReviews int

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.Contains(path, "reviews-") || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Parse city,country from folder
		relPath, _ := filepath.Rel(baseDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 2 {
			return nil
		}

		cityCountry := strings.SplitN(parts[0], ",", 2)
		if len(cityCountry) != 2 {
			return nil
		}
		//city := strings.TrimSpace(cityCountry[0])
		//country := strings.TrimSpace(cityCountry[1])

		// Parse filename: reviews-hotelname-hotelid.html
		filename := strings.TrimSuffix(filepath.Base(path), ".html")
		lastDash := strings.LastIndex(filename, "-")
		if lastDash == -1 {
			log.Printf("Invalid filename: %s", filename)
			return nil
		}

		hotelID := filename[lastDash+1:]
		log.Printf("Processing file for HotelID: %s", hotelID)

		reviews := extractYelpReviewsFromSavedHTML(path)

		for _, r := range reviews {
			hash := sha256.New()
			hash.Write([]byte(r["Review"] + r["Name"]))
			reviewID := hex.EncodeToString(hash.Sum(nil))

			rating, _ := strconv.ParseFloat(r["Rating"], 64)

			review := &models.HotelReview{
				HotelID:        hotelID,
				Source:         "yelp",
				SourceReviewID: reviewID,
				ReviewerName:   r["Name"],
				ReviewText:     r["Review"],
				Rating:         rating,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			if err := db.WithContext(ctx).
				Where("source_review_id = ? AND source = ?", reviewID, "yelp").
				FirstOrCreate(review).Error; err != nil {
				log.Printf("Failed to save review for %s: %v", hotelID, err)
			} else {
				totalReviews++
			}
		}

		log.Printf("Processed %s → %d reviews (HotelID: %s)", filename, len(reviews), hotelID)
		return nil
	})

	return totalReviews, err
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env: %v", err)
	}

	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	baseDir := "hotelReviewSaved/yelp"

	count, err := processYelpSavedHotelPages(db.DB, ctx, baseDir)
	if err != nil {
		log.Printf("Error during processing: %v", err)
	}

	log.Printf("Yelp review extraction completed. Total reviews saved: %d", count)
}
