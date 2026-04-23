//module example.com/p
//
//require (
//google.golang.org/api v0.271.0
//cloud.google.com/go/bigquery v1.74.0
//)

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

	type Hotel struct {
		Name     string
		ID       string
		OldPhoto string
	}

	var hotels []Hotel
	skipped := 0
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
		hotels = append(hotels, Hotel{
			Name:     fmt.Sprintf("%v", row["hotel_name"]),
			ID:       fmt.Sprintf("%v", row["hotel_id"]),
			OldPhoto: fmt.Sprintf("%v", row["photo_name"]),
		})
		//log.Printf("Hotel: %s, ID: %s, OldPhoto: %s\n", hotels[len(hotels)-1].Name, hotels[len(hotels)-1].ID, hotels[len(hotels)-1].OldPhoto[:20])
	}
	log.Printf("Found %d hotels to refresh. Skipped: %d\n", len(hotels), skipped)

	// For each hotel, fetch new photo from Google Places API
	for i, hotel := range hotels {
		url := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", hotel.ID)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			log.Printf("failed to create request for %d %s: %v\n", i, hotel.Name, err)
			continue
		}
		req.Header.Set("X-Goog-Api-Key", apiKey)
		req.Header.Set("X-Goog-FieldMask", "id,photos")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("failed to call API for %d %s: %v\n", i, hotel.Name, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("API returned %d for %d %s\n", resp.StatusCode, i, hotel.Name)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed to read response for %s (%d): %v", hotel.Name, i, err)
			continue
		}

		var placesResp PlacesResponse
		if err := json.Unmarshal(body, &placesResp); err != nil {
			log.Printf("failed to unmarshal response for %s (%d): %v\n", hotel.Name, i, err)
			continue
		}

		var newPhotoName string
		if len(placesResp.Photos) > 0 {
			newPhotoName = placesResp.Photos[0].Name
		}

		log.Printf("New photo for %d %s(%s): %s.  Old photo: %s\n", i, hotel.Name, hotel.ID, newPhotoName[len(newPhotoName)-20:], hotel.OldPhoto[len(hotel.OldPhoto)-20:])
		// Update reviews and embeddings
		updateReviews := fmt.Sprintf(`UPDATE %s r SET photo_name = @photo 
          WHERE hotel_id = @hotel_id`,
			tableReviews)
		q := bq.BQClient.Query(updateReviews)
		q.Parameters = []bigquery.QueryParameter{
			{Name: "photo", Value: newPhotoName},
			{Name: "hotel_id", Value: hotel.ID},
		}
		_, err = q.Run(ctx)
		if err != nil {
			log.Printf("failed to update reviews for %s(%s): %v", hotel.Name, hotel.ID, err)
			continue
		}

		// Update embeddings table
		updateEmbed := fmt.Sprintf(`UPDATE %s e SET e.photo_name = @photo 
          WHERE hotel_name = @hotel_name`,
			tableEmbed)
		q2 := bq.BQClient.Query(updateEmbed)
		q2.Parameters = []bigquery.QueryParameter{
			{Name: "photo", Value: newPhotoName},
			{Name: "hotel_name", Value: hotel.Name},
		}
		_, err = q2.Run(ctx)
		if err != nil {
			log.Printf("failed to update embeddings for %s: %v", hotel.Name, err)
			continue
		}

		log.Printf("Updated photo for %d %s", i, hotel.Name)
		if (i+1)%10 == 0 {
			log.Printf("Sleeping for 10 seconds to avoid rate limiting\n")
			time.Sleep(30 * time.Second)
		} else {
			time.Sleep(10 * time.Second)
		}
	}

	log.Printf("Refresh completed for %d hotels\n", len(hotels))

	return nil
}

func main() {
	RefreshPhotos()
}
