package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

var apiKey = os.Getenv("GOOGLE_PLACES_API_KEY")
var embeddings = "bigReview_embeddings"
var hotels = "bigHotels"
var reviews = "bigReviews"
var project = "golang1212025"
var dataset = "alpacaCentral"
var places_signature = os.Getenv("GOOGLE_PLACES_SIGNATURE")

type BQ struct {
	BQClient  *bigquery.Client
	ProjectID string
	DatasetID string
}
type PlacesResponse struct {
	ID     string `json:"id"`
	Photos []struct {
		Name string `json:"name"`
	} `json:"photos"`
}

type Hotel struct {
	Name     string
	ID       string
	OldPhoto string
}

func (bq *BQ) ExecuteQuery(ctx context.Context, query string, params []bigquery.QueryParameter) (*bigquery.RowIterator, error) {
	q := bq.BQClient.Query(query)
	q.Parameters = params

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}
	return it, nil
}

func (s *BQ) Close() error {
	return s.BQClient.Close()
}

func RefreshPhotos() {
	ctx := context.Background()

	// Load config (adjust path or use env vars)

	// Init BigQuery client
	bqClient, err := bigquery.NewClient(ctx, project)
	if err != nil {
		log.Printf("BQ client failed: %v", err)
		return
	}
	defer bqClient.Close()

	bq := &BQ{
		BQClient:  bqClient,
		ProjectID: project,
		DatasetID: dataset,
	}

	// Run the refresh logic (same as in queries.go)
	err = bq.RefreshPhotoNames(ctx)
	if err != nil {
		log.Printf("Refresh failed: %v", err)
		return
	}

	return
}

func (bq *BQ) RefreshPhotoNames(ctx context.Context) error {
	tableEmbed := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, embeddings)
	tableHotels := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, hotels)
	tableReviews := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, reviews)

	sql := fmt.Sprintf(`
		SELECT DISTINCT h.name as hotel_name, r.hotel_id, r.photo_name
		FROM %s e
		JOIN %s h ON h.name = e.hotel_name
		JOIN %s r ON h.source_hotel_id = r.hotel_id
		WHERE r.hotel_id IS NOT NULL AND r.hotel_id != '' AND r.source = 'google'
		AND r.photo_name = e.photo_name
	`, tableEmbed, tableHotels, tableReviews)

	it, err := bq.ExecuteQuery(ctx, sql, nil)
	if err != nil {
		return fmt.Errorf("failed to query hotels: %w\n", err)
	}

	var hotels []Hotel
	var hotelID = make(map[string]bool)
	var newPhotos = make(map[string]string)
	skipped := 0
	skippedDuplicateHotels := 0
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read hotel row: %w", err)
		}
		if row["photo_name"] == nil || row["photo_name"] == "" {
			//log.Printf("Skipping hotel %s (%s) - No photo name found\n", fmt.Sprintf("%v", row["hotel_name"]), fmt.Sprintf("%v", row["hotel_id"]))
			skipped++
			continue
		}

		hotelIDString := fmt.Sprintf("%v", row["hotel_id"])
		if _, exists := hotelID[hotelIDString]; exists {
			log.Println("Skipping duplicate hotel_id")
			skippedDuplicateHotels++
			continue
		}

		hotelID[hotelIDString] = true
		hotels = append(hotels, Hotel{
			Name:     fmt.Sprintf("%v", row["hotel_name"]),
			ID:       fmt.Sprintf("%v", row["hotel_id"]),
			OldPhoto: fmt.Sprintf("%v", row["photo_name"]),
		})
		//log.Printf("Hotel: %s, ID: %s, OldPhoto: %s\n", hotels[len(hotels)-1].Name, hotels[len(hotels)-1].ID, hotels[len(hotels)-1].OldPhoto[:20])
	}

	log.Printf("Found %d hotels to refresh. Skipped: %d empty photos and %d duplicate hotels\n", len(hotels), skipped, skippedDuplicateHotels)

	// For each hotel, fetch new photo from Google Places API
	for _, hotel := range hotels {

		placeResp, err := googlePlacesQuery(ctx, hotel, places_signature)

		if err != nil {
			log.Printf("Failed to get place for hotel: %v", err)
			continue
		}
		time.Sleep(100 * time.Millisecond)

		var newPhotoName string

		if len(placeResp.Photos) > 0 {
			newPhotoName = placeResp.Photos[0].Name
		}
		newPhotos[hotel.Name] = newPhotoName
	}

	// Batch updates to reduce too many DML BigQuery throttling
	batchSize := 500
	for i := 0; i < len(hotels); i += batchSize {
		end := i + batchSize
		if end > len(hotels) {
			end = len(hotels)
		}
		batch := hotels[i:end]

		var hotelNames []string
		var hotelIDs []string
		caseReviews := "CASE "
		caseEmbed := "CASE "
		for _, h := range batch {
			if newPhoto, ok := newPhotos[h.Name]; ok && newPhoto != "" {
				hotelNames = append(hotelNames, fmt.Sprintf("'%s'", strings.ReplaceAll(h.Name, "'", "\\'"))) // Escape quotes
				hotelIDs = append(hotelIDs, fmt.Sprintf("'%s'", h.ID))
				caseReviews += fmt.Sprintf("WHEN hotel_id = '%s' THEN '%s' ", h.ID, newPhoto)
				caseEmbed += fmt.Sprintf("WHEN hotel_name = '%s' THEN '%s' ", strings.ReplaceAll(h.Name, "'", "\\'"), newPhoto)
			}
		}
		caseReviews += "END"
		caseEmbed += "END"
		inNames := strings.Join(hotelNames, ", ")
		inIDs := strings.Join(hotelIDs, ", ")

		// Batch update reviews
		updateReviews := fmt.Sprintf("UPDATE %s SET photo_name = %s WHERE hotel_id IN (%s)", tableReviews, caseReviews, inIDs)
		log.Println(updateReviews)
		q := bq.BQClient.Query(updateReviews)
		job, err := q.Run(ctx)
		if err != nil {
			log.Printf("Failed to run batch update reviews: %v", err)
		} else {
			log.Printf("Batch updated reviews for %d hotels, job ID: %s", len(batch), job.ID())
		}

		// Batch update embeddings
		updateEmbed := fmt.Sprintf("UPDATE %s SET photo_name = %s WHERE hotel_name IN (%s)", tableEmbed, caseEmbed, inNames)
		log.Println(updateEmbed)
		q2 := bq.BQClient.Query(updateEmbed)
		job2, err := q2.Run(ctx)
		if err != nil {
			log.Printf("Failed to run batch update embeddings: %v", err)
		} else {
			log.Printf("Batch updated embeddings for %d hotels, job ID: %s", len(batch), job2.ID())
		}

		time.Sleep(10 * time.Second) // Throttle between batches
	}

	log.Printf("Refresh completed for %d hotels\n", len(hotels))

	return nil
}

func googlePlacesQuery(ctx context.Context, hotel Hotel, placesSignature string) (PlacesResponse, error) {

	url := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", hotel.ID)
	log.Println("URL:", url)

	//signedURL, err := google_places.SignURL(url, placesSignature)
	//log.Println("signedURL:", signedURL)

	//if err != nil {
	//	log.Printf("Error signing URL: %v", err)
	//	return PlacesResponse{}, err
	//}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("failed to create request for %s: %v\n", hotel.Name, err)
		return PlacesResponse{}, err
	}

	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", "id,photos")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to call API for %s: %v\n", hotel.Name, err)
		return PlacesResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("API returned %d for %s\n", resp.StatusCode, hotel.Name)
		return PlacesResponse{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response for %s: %v", hotel.Name, err)
		return PlacesResponse{}, err
	}

	var placesResp PlacesResponse
	if err := json.Unmarshal(body, &placesResp); err != nil {
		log.Printf("failed to unmarshal response for %s: %v\n", hotel.Name, err)
		return PlacesResponse{}, err
	}
	return placesResp, nil
}

func main() {
	RefreshPhotos()
}
