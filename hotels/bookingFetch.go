package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/chukiagosoftware/alpaca/models"
)

type bookingSearchResponse struct {
	Result []struct {
		HotelID     int64   `json:"hotel_id"`
		Name        string  `json:"hotel_name"`
		City        string  `json:"city"`
		CountryCode string  `json:"countrycode"`
		Address     string  `json:"address"`
		Zip         string  `json:"zip"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		ReviewScore float64 `json:"review_score"`
		URL         string  `json:"url"`
		Phone       string  `json:"phone"`
		Email       string  `json:"email"`
	} `json:"result"`
}

type bookingProvider struct {
	apiKey string
	client *http.Client
}

func newBookingProvider() *bookingProvider {
	return &bookingProvider{
		apiKey: os.Getenv("BOOKING_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *bookingProvider) getProviderName() string {
	return models.HotelSourceBooking
}

func (p *bookingProvider) isEnabled() bool {
	return p.apiKey != ""
}

func (p *bookingProvider) fetchHotels(ctx context.Context, location string) ([]*models.Hotel, error) {
	baseURL := "https://distribution-xml.booking.com/2.7/json/hotelAvailability"

	params := url.Values{}
	params.Set("city", location)
	params.Set("room1", "A")
	checkin := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	checkout := time.Now().AddDate(0, 0, 31).Format("2006-01-02")
	params.Set("checkin", checkin)
	params.Set("checkout", checkout)
	params.Set("rows", "100")

	requestURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.SetBasicAuth(p.apiKey, "")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var bookingResp bookingSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&bookingResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var hotels []*models.Hotel
	for _, result := range bookingResp.Result {
		lat := result.Latitude
		lng := result.Longitude
		rating := result.ReviewScore

		hotel := &models.Hotel{
			HotelID:       fmt.Sprintf("booking_%d", result.HotelID),
			Source:        models.HotelSourceBooking,
			SourceHotelID: strconv.FormatInt(result.HotelID, 10),
			Name:          result.Name,
			City:          result.City,
			Country:       result.CountryCode,
			StreetAddress: result.Address,
			PostalCode:    result.Zip,
			Latitude:      &lat,
			Longitude:     &lng,
			Phone:         result.Phone,
			Email:         result.Email,
			Website:       result.URL,
			BookingRating: &rating,
			LastUpdate:    time.Now().Format(time.RFC3339),
		}
		hotels = append(hotels, hotel)
	}

	log.Printf("Fetched %d hotels from Booking.com", len(hotels))
	return hotels, nil
}
