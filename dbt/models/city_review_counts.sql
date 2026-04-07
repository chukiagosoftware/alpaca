{{ config(materialized='table') }}

WITH reviews AS (
    SELECT
        hotel_id,
        reviewer_location,
        source_review_id,
        review_text,
        reviewer_name,
        review_date,
        rating
FROM {{ ref('all_reviews') }}
    ),

    hotels AS (
SELECT
    source_hotel_id,
    hotel_name,
    city as hotel_city,
    country as hotel_country
FROM {{ ref('all_hotels') }}
    ),

    city_mapped AS (
SELECT
    r.*,
    COALESCE(h.hotel_city,
    REGEXP_EXTRACT(r.reviewer_location, r'^[^,]+')) as city,
    COALESCE(h.hotel_country,
    REGEXP_EXTRACT(r.reviewer_location, r', ([^,]+)$')) as country
FROM reviews r
    LEFT JOIN hotels h ON h.source_hotel_id = r.hotel_id
    )

SELECT
    c.city,
    c.country,
    dc.continent,
    COUNT(*) as review_count,
    COUNT(DISTINCT c.hotel_id) as unique_hotels,
    COUNT(*) < 50 as is_low_review_city
FROM city_mapped c
         LEFT JOIN {{ ref('dim_cities') }} dc
ON TRIM(LOWER(dc.city)) = TRIM(LOWER(c.city))
GROUP BY c.city, c.country, dc.continent
ORDER BY review_count ASC