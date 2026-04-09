{{ config(materialized='table') }}

WITH cities_base AS (
    SELECT city, country, continent FROM {{ ref('dim_cities') }}
    ),

    review_stats AS (
SELECT
    COALESCE(h.city, REGEXP_EXTRACT(r.reviewer_location, r'^[^,]+')) as city,
    COUNT(*) as review_count,
    COUNT(DISTINCT r.hotel_id) as unique_hotels
FROM {{ ref('all_reviews') }} r
    LEFT JOIN {{ ref('all_hotels') }} h ON h.source_hotel_id = r.hotel_id
GROUP BY 1
    ),

    hotel_stats AS (
SELECT
    city,
    COUNT(DISTINCT source_hotel_id) as hotel_count
FROM {{ ref('all_hotels') }}
GROUP BY 1
    )

SELECT
    cb.city,
    cb.country,
    cb.continent,
    COALESCE(r.review_count, 0) as review_count,
    COALESCE(h.hotel_count, 0) as hotel_count,
    CASE WHEN COALESCE(r.review_count, 0) = 0 THEN TRUE ELSE FALSE END as zero_reviews,
    CASE WHEN COALESCE(r.review_count, 0) < 50 THEN TRUE ELSE FALSE END as low_reviews,
    CASE WHEN COALESCE(h.hotel_count, 0) = 0 THEN TRUE ELSE FALSE END as zero_hotels,
    CASE WHEN COALESCE(h.hotel_count, 0) < 50 THEN TRUE ELSE FALSE END as low_hotels
FROM cities_base cb
         LEFT JOIN review_stats r ON TRIM(LOWER(cb.city)) = TRIM(LOWER(r.city))
         LEFT JOIN hotel_stats h ON TRIM(LOWER(cb.city)) = TRIM(LOWER(h.city))
ORDER BY review_count ASC, hotel_count ASC