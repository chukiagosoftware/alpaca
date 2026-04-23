package main

import (
	"log"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/chukiagosoftware/alpaca/vertex"

	"strconv"
)

type LocationGroup struct {
	Continent     string   `json:"continent" bigquery:"continent"`
	CityCountries []string `json:"city_countries" bigquery:"city_countries"`
}

type PlacesResponse struct {
	ID     string `json:"id"`
	Photos []struct {
		Name string `json:"name"`
	} `json:"photos"`
}

type BQ struct {
	BQClient  *bigquery.Client
	ProjectID string
	DatasetID string
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

func NewBigQueryService(ctx context.Context, config vertex.Config) (*BQ, error) {

	log.Printf("DEBUG: Creating BigQuery client for project: %s", config.ProjectID)

	bqClient, err := bigquery.NewClient(ctx, config.ProjectID)
	if err != nil {
		log.Printf("DEBUG: BigQuery client creation failed with error: %v", err)
		return nil, err
	}
	log.Printf("DEBUG: BigQuery client created successfully")
	return &BQ{
		BQClient:  bqClient,
		ProjectID: config.ProjectID,
		DatasetID: config.DatasetID,
	}, nil
}

func (s *BQ) Close() error {
	return s.BQClient.Close()
}

func (bq *BQ) GetMetadataByIDs(ctx context.Context, vectorResults []vertex.VectorResult, config *vertex.Config) ([]map[string]any, error) {
	if len(vectorResults) == 0 {
		return nil, nil
	}

	// Extract IDs for the query
	ids := make([]int64, 0, len(vectorResults))
	idToDistance := make(map[string]float64, len(vectorResults))

	for _, vr := range vectorResults {
		if i, err := strconv.ParseInt(vr.ID, 10, 64); err == nil {
			ids = append(ids, i)
			idToDistance[vr.ID] = vr.Distance
		}
	}

	if len(ids) == 0 {
		return nil, nil
	}

	tableEmbed := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, config.BigReviewEmbeddings)
	//tableReviews := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, config.BigReviews)
	tableHotels := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, config.BigHotels)

	// Embedding ID is used to add the vector distance to results
	sql := fmt.Sprintf(`
		SELECT
		    e.id,
			e.review_text,
			e.rating,
			e.reviewer_name,
			e.google_maps_uri,
			e.photo_name,
			e.city,
			e.country,
			e.continent,
			e.hotel_name,
			h.street_address
		FROM %s e
		JOIN %s h ON h.name = e.hotel_name
		WHERE e.id IN UNNEST(@ids)
	`, tableEmbed, tableHotels)

	params := []bigquery.QueryParameter{
		{Name: "ids", Value: ids},
	}

	it, err := bq.ExecuteQuery(ctx, sql, params)
	if err != nil {
		return nil, fmt.Errorf("metadata query failed: %w", err)
	}

	metadataMap := make(map[string]map[string]any)
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		idStr := fmt.Sprintf("%v", row["id"])
		m := make(map[string]any)
		for k, v := range row {
			m[k] = v
		}
		metadataMap[idStr] = m
	}

	// Rebuild results in original vector search order, attaching distance
	var finalResults []map[string]any
	for _, vr := range vectorResults {
		if meta, ok := metadataMap[vr.ID]; ok {
			meta["distance"] = vr.Distance
			finalResults = append(finalResults, meta)
		}
	}

	return finalResults, nil
}

// GetDistinctLocations returns sorted distinct continents/countries/cities from bigReviews_embeddings.
func (bq *BQ) GetDistinctLocations(ctx context.Context) ([]LocationGroup, error) {
	richEmbeddedReviews := fmt.Sprintf("%s.%s.bigReview_embeddings", bq.ProjectID, bq.DatasetID)

	// cannot use alias defined in select in the where clause but can define it for output labelling
	sql := fmt.Sprintf(`
			SELECT continent,
    				ARRAY_AGG(DISTINCT CONCAT(city, ', ', country) ORDER BY CONCAT(city, ', ', country)) as city_countries
			FROM %s
			WHERE continent IS NOT NULL 
			  AND continent != ''
			  AND city IS NOT NULL 
			  AND city != ''
			  AND country IS NOT NULL 
			  AND country != ''
			GROUP BY continent
			ORDER BY continent`,
		richEmbeddedReviews)

	params := []bigquery.QueryParameter{
		//{
		//	Name: "continents",
		//	Value: []string{
		//		"USA",
		//		"mexico",
		//		"canada",
		//		"caribbean",
		//		"centralAmerica",
		//		"southamerica",
		//		"oceania",
		//		"europe",
		//		"asia",
		//		"africa",
		//	},
		//},
	}

	it, err := bq.ExecuteQuery(ctx, sql, params)
	if err != nil {
		return nil, err
	}
	//defer it.Close()

	var groups []LocationGroup
	for {
		var g LocationGroup
		err := it.Next(&g)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// RefreshPhotoNames refreshes expired photo names for hotels by querying Google Places API and updating both bigReviewEmbeddings and main_hotel_reviews_all_110k tables.
// RefreshPhotoNames refreshes expired photo names for hotels by querying Google Places API and updating both bigReviewEmbeddings and main_hotel_reviews_all_110k tables.
func (bq *BQ) RefreshPhotoNames(ctx context.Context, config *vertex.Config) error {
	tableEmbed := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, config.BigReviewEmbeddings)
	tableHotels := fmt.Sprintf("%s.%s.%s", bq.ProjectID, bq.DatasetID, config.BigHotels)

	// Query distinct hotel_name and hotel_id from the join
	sql := fmt.Sprintf(`
		SELECT DISTINCT h.name as hotel_name, h.hotel_id
		FROM %s e
		JOIN %s h ON h.name = e.hotel_name
		WHERE h.hotel_id IS NOT NULL AND h.hotel_id != ''
	`, tableEmbed, tableHotels)

	it, err := bq.ExecuteQuery(ctx, sql, nil)
	if err != nil {
		return fmt.Errorf("failed to query hotels: %w", err)
	}

	type Hotel struct {
		Name string
		ID   string
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
			Name: fmt.Sprintf("%v", row["hotel_name"]),
			ID:   fmt.Sprintf("%v", row["hotel_id"]),
		})
	}

	// For each hotel, fetch new photo from Google Places API
	for _, hotel := range hotels {
		url := fmt.Sprintf("https://places.googleapis.com/v1/places/%s?fields=id,photos", hotel.ID)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			log.Printf("failed to create request for %s: %v", hotel.Name, err)
			continue
		}
		req.Header.Set("X-Goog-Api-Key", config.GooglePlacesAPIKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("failed to call API for %s: %v", hotel.Name, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("API returned %d for %s", resp.StatusCode, hotel.Name)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed to read response for %s: %v", hotel.Name, err)
			continue
		}

		var placesResp PlacesResponse
		if err := json.Unmarshal(body, &placesResp); err != nil {
			log.Printf("failed to unmarshal response for %s: %v", hotel.Name, err)
			continue
		}

		var newPhotoName string
		if len(placesResp.Photos) > 0 {
			newPhotoName = placesResp.Photos[0].Name
		}

		// Update hotels table
		updateHotels := fmt.Sprintf("UPDATE %s SET photo_name = @photo WHERE name = @name", tableHotels)
		q := bq.BQClient.Query(updateHotels)
		q.Parameters = []bigquery.QueryParameter{
			{Name: "photo", Value: newPhotoName},
			{Name: "name", Value: hotel.Name},
		}
		_, err = q.Run(ctx)
		if err != nil {
			log.Printf("failed to update hotels for %s: %v", hotel.Name, err)
			continue
		}

		// Update embeddings table
		updateEmbed := fmt.Sprintf("UPDATE %s SET photo_name = @photo WHERE hotel_name = @name", tableEmbed)
		q2 := bq.BQClient.Query(updateEmbed)
		q2.Parameters = []bigquery.QueryParameter{
			{Name: "photo", Value: newPhotoName},
			{Name: "name", Value: hotel.Name},
		}
		_, err = q2.Run(ctx)
		if err != nil {
			log.Printf("failed to update embeddings for %s: %v", hotel.Name, err)
			continue
		}

		log.Printf("Updated photo for %s", hotel.Name)
	}

	return nil
}
