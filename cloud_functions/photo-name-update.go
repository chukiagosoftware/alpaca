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
	`, tableEmbed, tableHotels, tableReviews)

	it, err := bq.ExecuteQuery(ctx, sql, nil)
	if err != nil {
		return fmt.Errorf("failed to query hotels: %w", err)
	}

	type Hotel struct {
		Name     string
		ID       string
		OldPhoto string
	}

	var hotels []Hotel
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read hotel row: %w", err)
		}
		hotels = append(hotels, Hotel{
			Name:     fmt.Sprintf("%v", row["hotel_name"]),
			ID:       fmt.Sprintf("%v", row["hotel_id"]),
			OldPhoto: fmt.Sprintf("%v", row["photo_name"]),
		})
		log.Printf("Hotel: %s, ID: %s, OldPhoto: %s", hotels[len(hotels)-1].Name, hotels[len(hotels)-1].ID, hotels[len(hotels)-1].OldPhoto)
	}

	// For each hotel, fetch new photo from Google Places API
	for _, hotel := range hotels {
		url := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", hotel.ID)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			log.Printf("failed to create request for %s: %v\n", hotel.Name, err)
			continue
		}
		req.Header.Set("X-Goog-Api-Key", apiKey)
		req.Header.Set("X-Goog-FieldMask", "id,photos")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("failed to call API for %s: %v\n", hotel.Name, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("API returned %d for %s\n", resp.StatusCode, hotel.Name)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed to read response for %s: %v", hotel.Name, err)
			continue
		}

		var placesResp PlacesResponse
		if err := json.Unmarshal(body, &placesResp); err != nil {
			log.Printf("failed to unmarshal response for %s: %v\n", hotel.Name, err)
			continue
		}

		var newPhotoName string
		if len(placesResp.Photos) > 0 {
			newPhotoName = placesResp.Photos[0].Name
		}

		log.Printf("New photo for %s(%s): %s.  Old photo: %s\n", hotel.Name, hotel.ID, newPhotoName, hotel.OldPhoto)
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

		log.Printf("Updated photo for %s", hotel.Name)
	}

	// Verify

	verifySQL := fmt.Sprintf(`
    SELECT r.hotel_name, r.hotel_id, r.photo_name
    FROM %s r
    WHERE r.hotel_id IN UNNEST(@hotel_ids) AND r.source = 'google'
`, tableReviews) // Use reviews since it's the primary table with hotel_id

	var hotelIDs []string
	for _, h := range hotels {
		hotelIDs = append(hotelIDs, h.ID)
	}

	qVerify := bq.BQClient.Query(verifySQL)
	qVerify.Parameters = []bigquery.QueryParameter{
		{Name: "hotel_ids", Value: hotelIDs},
	}
	itVerify, err := qVerify.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to run verification query: %w", err)
	}

	updatedHotels := make(map[string]string) // hotel_id -> new photo_name
	for {
		var row map[string]bigquery.Value
		err := itVerify.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read verification row: %w", err)
		}
		id := fmt.Sprintf("%v", row["hotel_id"])
		photo := fmt.Sprintf("%v", row["photo_name"])
		updatedHotels[id] = photo
	}

	// Compare and log results
	for _, hotel := range hotels {
		newPhoto, exists := updatedHotels[hotel.ID]
		if !exists {
			log.Printf("Verification failed: No updated data found for hotel %s (%s)", hotel.Name, hotel.ID)
			continue
		}
		if newPhoto != hotel.OldPhoto {
			log.Printf("SUCCESS: Photo updated for %s (%s) - Old: %s, New: %s", hotel.Name, hotel.ID, hotel.OldPhoto, newPhoto)
		} else {
			log.Printf("NO CHANGE: Photo for %s (%s) remains %s (check if API returned new data or if update conditions matched)", hotel.Name, hotel.ID, hotel.OldPhoto)
		}
	}

	return nil
}

func main() {
	RefreshPhotos()
}
