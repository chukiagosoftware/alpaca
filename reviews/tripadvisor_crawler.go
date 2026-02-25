package main

import (
	"context"
	"log"

	"github.com/chukiagosoftware/alpaca/models"
)

// TripadvisorCrawler crawls reviews from Tripadvisor
type TripadvisorCrawler struct{}

func NewTripadvisorCrawler() *TripadvisorCrawler {
	return &TripadvisorCrawler{}
}

func (c *TripadvisorCrawler) GetSourceName() string {
	return models.SourceTripadvisor
}

func (c *TripadvisorCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Tripadvisor web scraping or API integration
	log.Printf("Tripadvisor crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}
