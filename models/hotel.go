package models

import (
	"gorm.io/gorm"
    "gorm.io/datatypes"
	"time"
)


type HotelAmadeusOauth2 struct {
	
    Type string `json:"type"`
    Username string `json:"username"`
	Password string `json:"password"`
    Application_name string `json:"application_name"`
    Client_id string `json:"client_id"`
    Token_type string `json:"token_type"`
    Access_token string `json:"access_token"`
    Expires_in int64 `json:"expires_in"`
	Expires_at int64 `json:"expires_at"`
    State string `json:"state"`
	Grant_type string `json:"grant_type"`
    Scope string `json:"scope"`

}

// HotelsListResponse represents the full API response with data and meta fields.
type HotelsListResponse struct {
	Data []HotelAPIItem `json:"data"`
	Meta HotelsListMeta `json:"meta"`
}

// HotelAPIItem represents a single hotel item as returned by the API.
type HotelAPIItem struct {
	gorm.Model
	ChainCode string        `json:"chainCode"`
	IATACode  string        `json:"iataCode"`
	DupeID    int64         `json:"dupeId"`
	Name      string        `json:"name"`
	HotelID   string        `json:"hotelId"`
	GeoCode   datatypes.JSON  `json:"geoCode"`
	Address   datatypes.JSON  `json:"address"`
	Distance  datatypes.JSON  `json:"distance"`
}

// HotelGeoCode represents the latitude and longitude of a hotel.
type HotelGeoCode struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// HotelAddress represents the address details of a hotel.
type HotelAddress struct {
	CountryCode string `json:"countryCode"`
}

// HotelDistance represents the distance information of a hotel.
type HotelDistance struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// HotelsListMeta captures pagination and other metadata from the API response.
type HotelsListMeta struct {
	
    Count int                `json:"count"`
	Links HotelsListMetaLink `json:"links"`
	Next  string              `json:"next,omitempty"` // Optional field for next page link
	Last string            `json:"last,omitempty"` // Optional field for previous page link
}

// HotelsListMetaLink captures the links object in the meta field.
type HotelsListMetaLink struct {
	Self string `json:"self"`
}

// ExternalLink represents an external link related to a hotel.
type ExternalLink struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Icon        string    `json:"icon"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Rating      int64     `json:"rating"`
	HotelId     int64     `json:"hotelId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Hotel represents a hotel with its details, external links, and reviews.
type Hotel struct {
	ChainCode       string        `json:"chainCode"`
	IATACode        string        `json:"iataCode"`
	DupeID          int64         `json:"dupeId"`
	HotelID         string        `json:"hotelId"`
	GeoCode         HotelGeoCode  `json:"geoCode"`
	Distance        HotelDistance `json:"distance"`
	Address         string        `json:"address"`
	City            string        `json:"city"`
	Region          string        `json:"region"`
	Country         string        `json:"country"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Warnings        []string      `json:"warnings"`
	Bonus           []string      `json:"bonus"`
	Quiet           bool          `json:"quiet"`
	ExternalLinkIds []int64       `json:"externalLinkIds"`
	Tags            []string      `json:"tags"`
	PriceRange      string        `json:"price_range"`
}

// Review represents a review for a hotel.
type Review struct {
	HotelId   int64  `json:"hotelId"`
	Rating    int64  `json:"rating"`
	Comment   string `json:"comment"`
	Commenter string `json:"commenter"`
}
