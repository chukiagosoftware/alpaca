; deduplicate and set zero values

CREATE OR REPLACE TABLE `golang1212025.alpaca.hotels`
AS
SELECT
    ID,
    hotel_id,
    source,
    source_hotel_id,
    TRIM(name) AS name,
    TRIM(City) AS City,
    TRIM(Country) AS Country,
    COALESCE(Latitude, 0.0) AS Latitude,
    COALESCE(Longitude, 0.0) AS Longitude,
    TRIM(street_address) AS street_address,
    TRIM(postal_code) AS postal_code,
    TRIM(Phone) AS Phone,
    TRIM(Website) AS Website,
    TRIM(Email) AS Email,
    COALESCE(amadeus_rating, 0.0) AS amadeus_rating,
    COALESCE(google_rating, 0.0) AS google_rating,
    COALESCE(Recommended, FALSE) AS Recommended,
    COALESCE(admin_flag, FALSE) AS admin_flag,
    COALESCE(Quality, FALSE) AS Quality,
    COALESCE(Quiet, FALSE) AS Quiet,
    TRIM(important_note) AS important_note,
    TRIM(Type) AS Type,
    COALESCE(DupeID, 0) AS DupeID,
    TRIM(iata_code) AS iata_code,
    address_json,
    geo_code_json,
    distance_json,
    TRIM(LastUpdate) AS LastUpdate,
    CreatedAt,
    UpdatedAt,
    TRIM(state_code) AS state_code,
    COALESCE(number_of_reviews, 0) AS number_of_reviews,
    COALESCE(number_of_ratings, 0) AS number_of_ratings,
    COALESCE(overall_rating, 0.0) AS overall_rating,
    TRIM(Sentiments) AS Sentiments
FROM `golang1212025.alpaca.hotels`
         QUALIFY
    ROW_NUMBER() OVER (PARTITION BY hotel_id, name, City, Country ORDER BY ID) = 1



