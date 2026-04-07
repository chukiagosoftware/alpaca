{{ config(materialized='table') }}

WITH latest_reviews AS (
    SELECT
        id,
        hotel_id,
        source_review_id,
        insert_id,
        review_text,
        reviewer_name,
        rating,
        review_date,
        google_maps_uri,
        photo_name,
        table_source
FROM {{ ref('all_reviews') }}
    ),

    embeddings AS (
SELECT id, hotel_name
FROM {{ source('bq', 'bigReview_embeddings') }}

UNION DISTINCT

SELECT id, hotel_name
FROM {{ source('bq', 'review_embeddings') }}
    )

SELECT
    r.*,
    h.hotel_name as hotel_hotel_name,
    h.street_address,
    h.city as hotel_city,
    h.country as hotel_country,
    c.city,
    c.country as city_country,
    c.continent
FROM latest_reviews r
         LEFT JOIN embeddings e
                   ON r.id = e.id
         LEFT JOIN {{ ref('all_hotels') }} h
ON h.source_hotel_id = r.hotel_id
    LEFT JOIN {{ ref('dim_cities') }} c
    ON TRIM(LOWER(c.city)) = h.city
WHERE e.id IS NULL
ORDER BY r.table_source, r.id