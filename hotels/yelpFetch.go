package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/chukiagosoftware/alpaca/models"
)

type yelpProvider struct {
	apiKey string
}

func newYelpProvider() *yelpProvider {
	return &yelpProvider{apiKey: os.Getenv("YELP_API_KEY")}
}

func (p *yelpProvider) getProviderName() string {
	return models.HotelSourceYelp
}

func (p *yelpProvider) isEnabled() bool {
	return p.apiKey != ""
}

func (p *yelpProvider) fetchHotels(ctx context.Context, location string) ([]*models.Hotel, error) {
	u, _ := url.Parse("https://api.yelp.com/v3/businesses/search")
	q := u.Query()
	q.Set("term", "hotels")
	q.Set("location", location)
	q.Set("limit", "50")
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Yelp API error: %d", resp.StatusCode)
	}

	var yelpResp struct {
		Businesses []struct {
			Id       string  `json:"id"`
			Name     string  `json:"name"`
			Rating   float64 `json:"rating"`
			Location struct {
				Address1 string `json:"address1"`
				City     string `json:"city"`
				State    string `json:"state"`
				ZipCode  string `json:"zip_code"`
				Country  string `json:"country"`
			} `json:"location"`
			Coordinates struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"coordinates"`
			Phone string `json:"phone"`
		} `json:"businesses"`
	}

	json.NewDecoder(resp.Body).Decode(&yelpResp)

	var hotels []*models.Hotel
	for _, b := range yelpResp.Businesses {
		hotel := &models.Hotel{
			HotelID:       fmt.Sprintf("yelp_%s", b.Id),
			Source:        models.HotelSourceYelp,
			SourceHotelID: b.Id,
			Name:          b.Name,
			City:          b.Location.City,
			Country:       b.Location.Country,
			Latitude:      &b.Coordinates.Latitude,
			Longitude:     &b.Coordinates.Longitude,
			StreetAddress: b.Location.Address1,
			PostalCode:    b.Location.ZipCode,
			Phone:         b.Phone,
			YelpRating:    b.Rating,
		}

		hotels = append(hotels, hotel)
	}

	log.Printf("Fetched %d hotels from Yelp", len(hotels))
	return hotels, nil
}
