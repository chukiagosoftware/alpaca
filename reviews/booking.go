package main

import (
	"context"
	"log"

	"github.com/chukiagosoftware/alpaca/models"
)

// BookingCrawler crawls reviews from Booking.com
type BookingCrawler struct{}

func NewBookingCrawler() *BookingCrawler {
	return &BookingCrawler{}
}

func (c *BookingCrawler) GetSourceName() string {
	return models.SourceBooking
}

func (c *BookingCrawler) CrawlReviews(ctx context.Context, hotel *models.Hotel) ([]*models.HotelReview, error) {
	// TODO: Implement Booking.com API or web scraping
	log.Printf("Booking crawler not yet implemented for hotel %s", hotel.Name)
	return []*models.HotelReview{}, nil
}
