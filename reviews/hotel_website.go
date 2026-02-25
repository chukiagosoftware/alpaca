package main

import (
	"context"
	"log"

	"github.com/chukiagosoftware/alpaca/models"
)

// HotelWebsiteCrawler crawls reviews from hotel's own website
type HotelWebsiteCrawler struct{}

func NewHotelWebsiteCrawler() *HotelWebsiteCrawler {
	return &HotelWebsiteCrawler{}
}

func (c *HotelWebsiteCrawler) GetSourceName() string {
	return models.SourceHotelWebsite
}

func (c *HotelWebsiteCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	if hotel.Website == "" {
		return []*models.HotelReview{}, nil
	}

	log.Printf("Hotel website crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}
