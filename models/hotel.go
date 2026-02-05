package models

import (
	"encoding/json"
	"time"
)

// HotelAmadeusOauth2 represents the OAuth2 token response from Amadeus API
type HotelAmadeusOauth2 struct {
	Type         string `json:"type"`
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	GrantType    string `json:"grant_type"`
	Scope        string `json:"scope"`
}

// HotelsListResponse represents the full API response with data and meta fields
type HotelsListResponse struct {
	Data []HotelAPIItem `json:"data"`
	Meta HotelsListMeta `json:"meta"`
}

// HotelAPIItem represents a single hotel item as returned by the Amadeus Hotel List API
type HotelAPIItem struct {
	ID        int64     `json:"-"`
	Type      string    `json:"type"`
	HotelID   string    `json:"hotelId"`
	ChainCode string    `json:"chainCode"`
	DupeID    int64     `json:"dupeId"`
	Name      string    `json:"name"`
	IATACode  string    `json:"iataCode"`
	Address   json.RawMessage `json:"address"`
	GeoCode   json.RawMessage `json:"geoCode"`
	Distance  json.RawMessage `json:"distance"`
	LastUpdate string   `json:"lastUpdate"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// HotelsListMeta captures pagination and other metadata from the API response
type HotelsListMeta struct {
	Count int                `json:"count"`
	Links HotelsListMetaLink `json:"links"`
}

// HotelsListMetaLink captures the links object in the meta field
type HotelsListMetaLink struct {
	Self  string `json:"self"`
	First string `json:"first"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
	Last  string `json:"last"`
}

// HotelSearchResponse represents the response from the Hotel Search API
type HotelSearchResponse struct {
	Data []HotelSearchData `json:"data"`
	Meta HotelSearchMeta   `json:"meta"`
}

// HotelSearchData represents detailed hotel information from the search API
type HotelSearchData struct {
	ID            int64     `json:"-"`
	Type          string    `json:"type"`
	HotelID       string    `json:"hotelId"`
	ChainCode     string    `json:"chainCode"`
	DupeID        int64     `json:"dupeId"`
	Name          string    `json:"name"`
	Rating        int       `json:"rating"`
	OfficialRating int      `json:"officialRating"`
	Description   json.RawMessage `json:"description"`
	Media         json.RawMessage `json:"media"`
	Amenities     json.RawMessage `json:"amenities"`
	Address       json.RawMessage `json:"address"`
	Contact       json.RawMessage `json:"contact"`
	Policies      json.RawMessage `json:"policies"`
	Available     bool      `json:"available"`
	Offers        json.RawMessage `json:"offers"`
	Self          string    `json:"self"`
	HotelDistance json.RawMessage `json:"hotelDistance"`
	LastUpdate    string    `json:"lastUpdate"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
}

// HotelSearchMeta represents metadata for the search response
type HotelSearchMeta struct {
	Count    int                  `json:"count"`
	Links    HotelSearchMetaLinks `json:"links"`
	Warnings []HotelSearchWarning `json:"warnings,omitempty"`
}

// HotelSearchMetaLinks represents links in the search metadata
type HotelSearchMetaLinks struct {
	Self string `json:"self"`
}

// HotelSearchWarning represents warnings in the search response
type HotelSearchWarning struct {
	Code   int    `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// HotelRatingsResponse represents the response from the Hotel Ratings API
type HotelRatingsResponse struct {
	Data     []HotelRatingsData    `json:"data"`
	Meta     HotelRatingsMeta      `json:"meta"`
	Warnings []HotelRatingsWarning `json:"warnings,omitempty"`
}

// HotelRatingsData represents detailed ratings information
type HotelRatingsData struct {
	ID              int64     `json:"-"`
	Type            string    `json:"type"`
	HotelID         string    `json:"hotelId"`
	NumberOfReviews int       `json:"numberOfReviews"`
	NumberOfRatings int       `json:"numberOfRatings"`
	OverallRating   int       `json:"overallRating"`
	Sentiments      json.RawMessage `json:"sentiments"`
	LastUpdate      string    `json:"lastUpdate"`
	CreatedAt       time.Time `json:"-"`
	UpdatedAt       time.Time `json:"-"`
}

// HotelRatingsSentiments represents the sentiment categories
type HotelRatingsSentiments struct {
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

// HotelRatingsMeta represents metadata for the ratings response
type HotelRatingsMeta struct {
	Count int                   `json:"count"`
	Links HotelRatingsMetaLinks `json:"links"`
}

// HotelRatingsMetaLinks represents links in the ratings metadata
type HotelRatingsMetaLinks struct {
	Self string `json:"self"`
}

// HotelRatingsWarning represents warnings in the ratings response
type HotelRatingsWarning struct {
	Code   int                       `json:"code"`
	Title  string                    `json:"title"`
	Detail string                    `json:"detail"`
	Source HotelRatingsWarningSource `json:"source"`
}

// HotelRatingsWarningSource represents the source of a warning
type HotelRatingsWarningSource struct {
	Parameter string `json:"parameter"`
	Pointer   string `json:"pointer"`
}

// InvalidHotelSearchID stores hotel IDs that are invalid for the Search API
type InvalidHotelSearchID struct {
	ID        int64     `json:"-"`
	HotelID   string    `json:"hotelId"`
	CreatedAt time.Time `json:"-"`
}
