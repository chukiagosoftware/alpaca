package main

import (
	"context"
	"log"

	"github.com/chukiagosoftware/alpaca/models"
)

// BingCrawler crawls reviews from Bing
type BingCrawler struct{}

func NewBingCrawler() *BingCrawler {
	return &BingCrawler{}
}

func (c *BingCrawler) GetSourceName() string {
	return models.SourceBing
}

func (c *BingCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Bing search API integration
	log.Printf("Bing crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}
