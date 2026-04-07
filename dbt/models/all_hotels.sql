{{ config(materialized='view') }}

SELECT
    source_hotel_id,
     name as hotel_name,
    street_address,
    city,
    country,
    source,
    'bigHotels' as table_source
FROM {{ source('bq', 'bigHotels') }}

UNION ALL

SELECT
    source_hotel_id,
    name as hotel_name,
    street_address,
    city,
    country,
    source,
    'hotels' as table_source
FROM {{ source('bq', 'hotels') }}

UNION ALL

SELECT
    source_hotel_id,
    name as hotel_name,
    street_address,
    city,
    country,
    source,
    'hotels_old_amadeusdev' as table_source
FROM {{ source('bq', 'hotels_old_from_airportcities_amadeusdev_google_tripadvisor') }}