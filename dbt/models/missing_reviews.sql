{{ config(materialized='table') }}

SELECT
    r.id,
    r.hotel_id,
    r.source_review_id,
    r.insert_id,
    r.review_text,
    r.reviewer_name,
    r.rating,
    r.review_date,
    r.google_maps_uri,
    r.photo_name,
    r.table_source,
    h.hotel_name as hotel_hotel_name,
    h.street_address,
    h.city as hotel_city,
    h.country as hotel_country,
    c.city,
    c.country as city_country,
    c.continent,
    'TRULY_MISSING' as status
FROM {{ ref('all_reviews') }} r
LEFT JOIN {{ ref('all_hotels') }} h
ON h.source_hotel_id = r.hotel_id
    LEFT JOIN {{ ref('dim_cities') }} c
    ON TRIM(LOWER(c.city)) = TRIM(LOWER(COALESCE(h.city, r.reviewer_location)))
WHERE NOT EXISTS (
    SELECT 1
    FROM (
    SELECT DISTINCT MD5(TRIM(REGEXP_REPLACE(LOWER(review_text), r'\s+', ' '))) as review_text_hash
    FROM {{ source('bq', 'bigReview_embeddings') }}

    UNION DISTINCT

    SELECT DISTINCT MD5(TRIM(REGEXP_REPLACE(LOWER(review_text), r'\s+', ' '))) as review_text_hash
    FROM {{ source('bq', 'review_embeddings') }}
    ) e
    WHERE e.review_text_hash = r.review_text_hash
    )
ORDER BY r.review_text