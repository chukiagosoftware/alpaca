package main

import (
	"context"
	"log"

	"github.com/chukiagosoftware/alpaca/models"
)

// ExpediaCrawler crawls reviews from Expedia
type ExpediaCrawler struct{}

func NewExpediaCrawler() *ExpediaCrawler {
	return &ExpediaCrawler{}
}

func (c *ExpediaCrawler) GetSourceName() string {
	return models.SourceExpedia
}

func (c *ExpediaCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Expedia API or web scraping
	log.Printf("Expedia crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}
