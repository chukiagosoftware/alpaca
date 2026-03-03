package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/gocolly/colly"
)

var tripadvisorLocationIDs = map[string]string{
	"London,England":   "g186338-London_England",
	"Austin,USA":       "g30196-Austin_Texas",
	"Sao Paulo,Brazil": "g303631-Sao_Paulo_State_of_Sao_Paulo",
	"Helsinki,Finland": "g189934-Helsinki_Uusimaa",
	"Warsaw,Poland":    "g274856-Warsaw_Mazovia_Province_Central_Poland",
	"Miami,USA":        "g34438-Miami_Florida",
	"Madrid,Spain":     "g187514-Madrid",
	"Bangkok,Thailand": "g293916-Bangkok",
	"Denver,USA":       "g33388-Denver_Colorado",
	"Sydney,Australia": "g255060-Sydney_New_South_Wales",
}

// ScrapedHotel pairs a hotel model with its URL for temporary use
type ScrapedHotel struct {
	Hotel models.Hotel
	URL   string
}

// searchHotels scrapes hotels from the given TripAdvisor URL

// searchHotels scrapes hotels from the given TripAdvisor URL using chromedp for JS rendering
func searchHotels(hotelsURL string) []ScrapedHotel {
	var hotels []ScrapedHotel

	// Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.WindowSize(1920, 1080),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(hotelsURL),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.OuterHTML(`body`, &html, chromedp.ByQuery), // Capture body HTML after JS
	)
	if err != nil {
		log.Printf("Error loading page: %v", err)
		return hotels
	}

	// Debug: Print the HTML
	log.Printf("Loaded body HTML length: %d", len(html))
	log.Printf("Body HTML: %s", html) // Full HTML; may be large

	// For now, return empty hotels (since we're just checking HTML)
	return hotels
}

func searchHotelsHTML(hotelsURL string) []ScrapedHotel {
	var hotels []ScrapedHotel
	c := colly.NewCollector()

	c.OnHTML("li.stFPg", func(e *colly.HTMLElement) { // Selector for hotel list items
		href := e.ChildAttr("a[href*='/Hotel_Review-']", "href")
		if href == "" {
			return
		}

		// Extract hotel ID from href (e.g., -d3523356-)
		re := regexp.MustCompile(`-d(\d+)-`)
		matches := re.FindStringSubmatch(href)
		if len(matches) < 2 {
			return
		}
		id := matches[1]

		name := e.ChildText("h3")
		if name == "" {
			return
		}

		// Ensure full URL
		if !strings.HasPrefix(href, "http") {
			href = "https://www.tripadvisor.com" + href
		}

		hotels = append(hotels, ScrapedHotel{
			Hotel: models.Hotel{
				SourceHotelID: id,
				Name:          name,
			},
			URL: href,
		})
	})

	c.Visit(hotelsURL)
	c.Wait()
	return hotels
}

func getHotelReview(hotelURL string) string {
	var reviews []string
	c := colly.NewCollector()

	// Selector for review text
	c.OnHTML("#REVIEWS > div > div.aRZXW > div > div > div:nth-child(2) > div > div > div.mSOQy > div.FRFxD._u > div:nth-child(1) > div > div:nth-child(1) > div.FKRgy.f.e > div._c > div > div.fIrGe._T.bgMZj > div > span > div > span", func(e *colly.HTMLElement) {
		text := strings.TrimSpace(e.Text)
		if text != "" {
			reviews = append(reviews, text)
		}
	})

	c.Visit(hotelURL)
	c.Wait()
	return strings.Join(reviews, "\n")
}

func main() {
	// Initialize database
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Iterate over the map
	for key, locationID := range tripadvisorLocationIDs {
		// Parse city and country from key
		parts := strings.Split(key, ",")
		if len(parts) != 2 {
			log.Printf("Invalid key: %s", key)
			continue
		}
		cityName := parts[0]
		country := parts[1]

		hotelsURL := fmt.Sprintf("https://www.tripadvisor.com/Hotels-%s-Hotels.html", locationID)
		log.Printf("Processing %s, %s with URL: %s", cityName, country, hotelsURL)

		hotels := searchHotels(hotelsURL)
		log.Printf("Found %d hotels for %s, %s", len(hotels), cityName, country)
		// Wait 3 seconds after fetching hotels list for a city
		time.Sleep(3 * time.Second)

		for _, scraped := range hotels {
			// Set HotelID and SourceHotelID to the locationID
			hotelModel := &scraped.Hotel
			hotelModel.HotelID = locationID
			hotelModel.Source = models.HotelSourceTripadvisor
			hotelModel.SourceHotelID = locationID
			hotelModel.City = cityName
			hotelModel.Country = country
			hotelModel.LastUpdate = time.Now().Format(time.RFC3339)

			if err := db.CreateOrUpdateHotel(ctx, hotelModel); err != nil {
				log.Printf("Error saving hotel: %v", err)
				continue
			}

			// Wait 1 second after saving hotel
			time.Sleep(1 * time.Second)

			// Get review
			reviewText := getHotelReview(scraped.URL)
			if reviewText != "" {
				review := &models.HotelReview{
					HotelID:    hotelModel.HotelID,
					Source:     models.SourceTripadvisor,
					ReviewText: reviewText,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				if err := db.SaveReview(ctx, review); err != nil {
					log.Printf("Error saving review: %v", err)
				}
			}
			// Wait 2 seconds after processing each hotel review
			time.Sleep(2 * time.Second)
		}
		// Wait 5 seconds between processing different cities
		time.Sleep(5 * time.Second)
	}

	log.Println("TripAdvisor scraping completed.")
}
