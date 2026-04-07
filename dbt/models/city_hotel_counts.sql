{{ config(materialized='table') }}

SELECT
    COALESCE(h.city, c.city) as city,
    COALESCE(h.country, c.country) as country,
    c.continent,
    COUNT(DISTINCT h.source_hotel_id) as hotel_count,
    COUNT(DISTINCT r.hotel_id) as review_linked_hotel_count
FROM {{ ref('dim_cities') }} c
LEFT JOIN {{ ref('all_hotels') }} h
ON TRIM(LOWER(c.city)) = TRIM(LOWER(h.city))
    LEFT JOIN {{ ref('all_reviews') }} r
    ON h.source_hotel_id = r.hotel_id
GROUP BY 1,2,3
ORDER BY hotel_count ASC