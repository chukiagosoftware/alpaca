package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type HotelAmadeusOauth2 struct {
	Type             string `json:"type"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Application_name string `json:"application_name"`
	Client_id        string `json:"client_id"`
	Token_type       string `json:"token_type"`
	Access_token     string `json:"access_token"`
	Expires_in       int64  `json:"expires_in"`
	Expires_at       int64  `json:"expires_at"`
	State            string `json:"state"`
	Grant_type       string `json:"grant_type"`
	Scope            string `json:"scope"`
}

// HotelsListResponse represents the full API response with data and meta fields.
type HotelsListResponse struct {
	Data []HotelAPIItem `json:"data"`
	Meta HotelsListMeta `json:"meta"`
}

// HotelAPIItem represents a single hotel item as returned by the Amadeus Hotel List API.
type HotelAPIItem struct {
	gorm.Model
	Type       string         `json:"type" gorm:"column:type"`
	HotelID    string         `json:"hotelId" gorm:"column:hotel_id;index"`
	ChainCode  string         `json:"chainCode" gorm:"column:chain_code"`
	DupeID     int64          `json:"dupeId" gorm:"column:dupe_id"`
	Name       string         `json:"name" gorm:"column:name"`
	IATACode   string         `json:"iataCode" gorm:"column:iata_code"`
	Address    datatypes.JSON `json:"address" gorm:"column:address"`
	GeoCode    datatypes.JSON `json:"geoCode" gorm:"column:geo_code"`
	Distance   datatypes.JSON `json:"distance" gorm:"column:distance"`
	LastUpdate string         `json:"lastUpdate" gorm:"column:last_update"`
}

// HotelAddress represents the address details of a hotel.
type HotelAddress struct {
	Lines       []string `json:"lines"`
	PostalCode  string   `json:"postalCode"`
	CityName    string   `json:"cityName"`
	CountryCode string   `json:"countryCode"`
	StateCode   string   `json:"stateCode,omitempty"`
}

// HotelGeoCode represents the latitude and longitude of a hotel.
type HotelGeoCode struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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
	ChainCode       string         `json:"chainCode"`
	IATACode        string         `json:"iataCode"`
	DupeID          int64          `json:"dupeId"`
	HotelID         string         `json:"hotelId"`
	GeoCode         datatypes.JSON `json:"geoCode"`
	Distance        int64          `json:"distance"`
	Address         string         `json:"address"`
	City            string         `json:"city"`
	Region          string         `json:"region"`
	Country         string         `json:"country"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Warnings        []string       `json:"warnings"`
	Bonus           []string       `json:"bonus"`
	Quiet           bool           `json:"quiet"`
	ExternalLinkIds []int64        `json:"externalLinkIds"`
	Tags            []string       `json:"tags"`
	PriceRange      string         `json:"price_range"`
}

// Review represents a review for a hotel.
type Review struct {
	HotelId   int64  `json:"hotelId"`
	Rating    int64  `json:"rating"`
	Comment   string `json:"comment"`
	Commenter string `json:"commenter"`
}

// HotelSentimentResponse represents the full response from the sentiment API.
type HotelSentimentResponse struct {
	Data     []HotelSentimentData    `json:"data"`
	Meta     HotelSentimentMeta      `json:"meta"`
	Warnings []HotelSentimentWarning `json:"warnings"`
}

type HotelSentimentData struct {
	Type            string                   `json:"type"`
	NumberOfReviews int                      `json:"numberOfReviews"`
	NumberOfRatings int                      `json:"numberOfRatings"`
	HotelID         string                   `json:"hotelId"`
	OverallRating   int                      `json:"overallRating"`
	Sentiments      HotelSentimentCategories `json:"sentiments"`
}

type HotelSentimentCategories struct {
	SleepQuality     int `json:"sleepQuality,omitempty"`
	Service          int `json:"service,omitempty"`
	Facilities       int `json:"facilities,omitempty"`
	RoomComforts     int `json:"roomComforts,omitempty"`
	ValueForMoney    int `json:"valueForMoney,omitempty"`
	Catering         int `json:"catering,omitempty"`
	Location         int `json:"location,omitempty"`
	Internet         int `json:"internet,omitempty"`
	PointsOfInterest int `json:"pointsOfInterest,omitempty"`
	Staff            int `json:"staff,omitempty"`
}

type HotelSentimentMeta struct {
	Count int                     `json:"count"`
	Links HotelSentimentMetaLinks `json:"links"`
}

type HotelSentimentMetaLinks struct {
	Self string `json:"self"`
}

type HotelSentimentWarning struct {
	Code   int                         `json:"code"`
	Title  string                      `json:"title"`
	Detail string                      `json:"detail"`
	Source HotelSentimentWarningSource `json:"source"`
}

type HotelSentimentWarningSource struct {
	Parameter string `json:"parameter"`
	Pointer   string `json:"pointer"`
}

// RatingsAmadeus represents the sentiment data for storage in the ratings_amadeus table.
type RatingsAmadeus struct {
	gorm.Model
	HotelID          string `json:"hotelId" gorm:"index"`
	Type             string `json:"type"`
	NumberOfReviews  int    `json:"numberOfReviews"`
	NumberOfRatings  int    `json:"numberOfRatings"`
	OverallRating    int    `json:"overallRating"`
	SleepQuality     int    `json:"sleepQuality"`
	Service          int    `json:"service"`
	Facilities       int    `json:"facilities"`
	RoomComforts     int    `json:"roomComforts"`
	ValueForMoney    int    `json:"valueForMoney"`
	Catering         int    `json:"catering"`
	Location         int    `json:"location"`
	Internet         int    `json:"internet"`
	PointsOfInterest int    `json:"pointsOfInterest"`
	Staff            int    `json:"staff"`
}
