package main

import (
	"context"
	"log"

	"github.com/chukiagosoftware/alpaca/models"
)

// YelpCrawler crawls reviews from Yelp
type YelpCrawler struct{}

func NewYelpCrawler() *YelpCrawler {
	return &YelpCrawler{}
}

func (c *YelpCrawler) GetSourceName() string {
	return models.SourceYelp
}

func (c *YelpCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Yelp Fusion API integration
	log.Printf("Yelp crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}
