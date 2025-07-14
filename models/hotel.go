package models

import (
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
// This is the parent table that other hotel data references.
type HotelAPIItem struct {
	gorm.Model
	Type       string         `json:"type" gorm:"column:type"`
	HotelID    string         `json:"hotelId" gorm:"column:hotel_id;uniqueIndex;not null"`
	ChainCode  string         `json:"chainCode" gorm:"column:chain_code"`
	DupeID     int64          `json:"dupeId" gorm:"column:dupe_id"`
	Name       string         `json:"name" gorm:"column:name"`
	IATACode   string         `json:"iataCode" gorm:"column:iata_code"`
	Address    datatypes.JSON `json:"address" gorm:"column:address"`
	GeoCode    datatypes.JSON `json:"geoCode" gorm:"column:geo_code"`
	Distance   datatypes.JSON `json:"distance" gorm:"column:distance"`
	LastUpdate string         `json:"lastUpdate" gorm:"column:last_update"`

	// Relationships
	SearchData  *HotelSearchData  `json:"searchData,omitempty" gorm:"foreignKey:HotelID;references:HotelID"`
	RatingsData *HotelRatingsData `json:"ratingsData,omitempty" gorm:"foreignKey:HotelID;references:HotelID"`
}

// HotelsListMeta captures pagination and other metadata from the API response.
type HotelsListMeta struct {
	Count int                `json:"count"`
	Links HotelsListMetaLink `json:"links"`
}

// HotelsListMetaLink captures the links object in the meta field.
type HotelsListMetaLink struct {
	Self  string `json:"self"`
	First string `json:"first"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
	Last  string `json:"last"`
}

// ===== HOTEL SEARCH API MODELS =====

// HotelSearchResponse represents the response from the Hotel Search API
type HotelSearchResponse struct {
	Data []HotelSearchData `json:"data"`
	Meta HotelSearchMeta   `json:"meta"`
}

// HotelSearchData represents detailed hotel information from the search API
// This is a child table that references HotelAPIItem
type HotelSearchData struct {
	gorm.Model
	Type           string         `json:"type" gorm:"column:type"`
	HotelID        string         `json:"hotelId" gorm:"column:hotel_id;uniqueIndex;not null;foreignKey:HotelID;references:HotelID"`
	ChainCode      string         `json:"chainCode" gorm:"column:chain_code"`
	DupeID         int64          `json:"dupeId" gorm:"column:dupe_id"`
	Name           string         `json:"name" gorm:"column:name"`
	Rating         int            `json:"rating" gorm:"column:rating"`
	OfficialRating int            `json:"officialRating" gorm:"column:official_rating"`
	Description    datatypes.JSON `json:"description" gorm:"column:description"`
	Media          datatypes.JSON `json:"media" gorm:"column:media"`
	Amenities      datatypes.JSON `json:"amenities" gorm:"column:amenities"`
	Address        datatypes.JSON `json:"address" gorm:"column:address"`
	Contact        datatypes.JSON `json:"contact" gorm:"column:contact"`
	Policies       datatypes.JSON `json:"policies" gorm:"column:policies"`
	Available      bool           `json:"available" gorm:"column:available"`
	Offers         datatypes.JSON `json:"offers" gorm:"column:offers"`
	Self           string         `json:"self" gorm:"column:self"`
	HotelDistance  datatypes.JSON `json:"hotelDistance" gorm:"column:hotel_distance"`
	LastUpdate     string         `json:"lastUpdate" gorm:"column:last_update"`

	// Relationship back to parent
	Hotel *HotelAPIItem `json:"hotel,omitempty" gorm:"foreignKey:HotelID;references:HotelID"`
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

// ===== HOTEL RATINGS API MODELS =====

// HotelRatingsResponse represents the response from the Hotel Ratings API
type HotelRatingsResponse struct {
	Data     []HotelRatingsData    `json:"data"`
	Meta     HotelRatingsMeta      `json:"meta"`
	Warnings []HotelRatingsWarning `json:"warnings,omitempty"`
}

// HotelRatingsData represents detailed ratings information
// This is a child table that references HotelAPIItem
type HotelRatingsData struct {
	gorm.Model
	Type            string         `json:"type" gorm:"column:type"`
	HotelID         string         `json:"hotelId" gorm:"column:hotel_id;uniqueIndex;not null;foreignKey:HotelID;references:HotelID"`
	NumberOfReviews int            `json:"numberOfReviews" gorm:"column:number_of_reviews"`
	NumberOfRatings int            `json:"numberOfRatings" gorm:"column:number_of_ratings"`
	OverallRating   int            `json:"overallRating" gorm:"column:overall_rating"`
	Sentiments      datatypes.JSON `json:"sentiments" gorm:"column:sentiments"`
	LastUpdate      string         `json:"lastUpdate" gorm:"column:last_update"`

	// Relationship back to parent
	Hotel *HotelAPIItem `json:"hotel,omitempty" gorm:"foreignKey:HotelID;references:HotelID"`
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
// and should be skipped in future runs.
type InvalidHotelSearchID struct {
	ID      uint   `gorm:"primaryKey"`
	HotelID string `gorm:"uniqueIndex;not null"`
}
