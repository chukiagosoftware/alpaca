package main

import (
	"context"
	"log"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
	"github.com/chukiagosoftware/alpaca/vertex"
)

func main() {

	config, err := vertex.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Loaded config: ProjectID=%s\n, Location=%s\n, DatasetID=%s\n, IndexID=%s\n, EndpointID=%s\n, Domain=%s\n",
		config.ProjectID,
		config.Location,
		config.DatasetID,
		config.IndexID,
		config.EndpointID,
		config.EndpointPublicDomainName)

	ctx := context.Background()
	s, err := vertex.NewBigQueryService(ctx, *config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// legacy first upload
	//bigTables := map[string]interface{}{
	//	"cities":  models.AirportCity{},
	//	"hotels":  models.Hotel{},

	// first bigtables
	bigTables := map[string]interface{}{
		//"bigReviews": models.HotelReviews{},
		//"bigHotels": models.Hotel{},
		"bigCity": models.City{},
	}

	for name, infer := range bigTables {
		if err := s.CreateBigQueryTable(ctx, infer, name); err != nil {
			log.Fatalf("Failed to create table %s: %v", name, err)
		}
	}

	// Upload data from local DB to BigQuery
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to local DB: %v", err)
	}
	defer db.Close()

	// Fetch and upload cities
	//var cities []models.AirportCity
	//if err := db.Find(&cities).Error; err != nil {
	//	log.Fatalf("Failed to fetch cities: %v", err)
	//}
	//if err := vertex.UploadData(ctx, s, "cities", cities); err != nil {
	//	log.Fatalf("Failed to upload cities: %v", err)
	//}
	//
	//// Fetch and upload hotels
	//var hotels []models.Hotel
	//if err := db.Find(&hotels).Error; err != nil {
	//	log.Fatalf("Failed to fetch hotels: %v", err)
	//}
	//if err := vertex.UploadData(ctx, s, "hotels", hotels); err != nil {
	//	log.Fatalf("Failed to upload hotels: %v", err)
	//}
	//log.Printf("Uploaded %d hotels", len(hotels))
	//
	// Old small batch
	//if err := vertex.UploadData(ctx, s, "reviews", reviews); err != nil {
	//	log.Fatalf("Failed to upload reviews: %v", err)
	//}

	//// Fetch and upload reviews
	//var reviews []models.HotelReview
	//if err := db.Find(&reviews).Error; err != nil {
	//	log.Fatalf("Failed to fetch reviews: %v", err)
	//}
	//
	//if err := vertex.UploadBatches(ctx, s, "bigReviews", reviews); err != nil {
	//	log.Printf("upload failed: %w", err)
	//}
	//
	//log.Printf("Uploaded %d reviews", len(reviews))

	//var hotels []models.Hotel
	//if err := db.Find(&hotels).Error; err != nil {
	//	log.Fatalf("Failed to fetch reviews: %v", err)
	//}

	var cities []models.City
	if err := db.Find(&cities).Error; err != nil {
		log.Fatalf("Failed to fetch cities: %v", err)
	}

	if err := vertex.UploadBatches(ctx, s, "bigCity", cities); err != nil {
		log.Printf("upload failed: %w", err)
	}

	log.Printf("Uploaded %d cities", len(cities))

	// ToDo complete the batchupload.go
	//gcsClient, err := storage.NewClient(ctx)
	//if err != nil { log.Fatal(err) }
	//defer gcsClient.Close()
	//
	//if err := UploadLoadBatches(ctx, s, gcsClient, "your-gcs-bucket", "bigReviews", reviews); err != nil {
	//	log.Fatal(err)
	//}

}
