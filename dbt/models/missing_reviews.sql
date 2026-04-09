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
        table_source,
        reviewer_location,
        MD5(TRIM(REGEXP_REPLACE(LOWER(review_text), r'\s+', ' '))) as review_text_hash
FROM {{ ref('all_reviews') }}
WHERE review_text IS NOT NULL
    ),

    embeddings AS (
SELECT
    MD5(TRIM(REGEXP_REPLACE(LOWER(review_text), r'\s+', ' '))) as review_text_hash,
    hotel_name,
    city,
    country,
    continent
FROM {{ source('bq', 'bigReview_embeddings') }}

UNION ALL

SELECT
    MD5(TRIM(REGEXP_REPLACE(LOWER(review_text), r'\s+', ' '))) as review_text_hash,
    hotel_name,
    city,
    country,
    NULL as continent
FROM {{ source('bq', 'review_embeddings') }}
    )

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
    'TRULY_MISSING' as missing_status
FROM latest_reviews r
         LEFT JOIN embeddings e ON r.review_text_hash = e.review_text_hash
         LEFT JOIN {{ ref('all_hotels') }} h ON h.source_hotel_id = r.hotel_id
    LEFT JOIN {{ ref('dim_cities') }} c
    ON TRIM(LOWER(c.city)) = TRIM(LOWER(COALESCE(h.city, r.reviewer_location)))
WHERE e.review_text_hash IS NULL
ORDER BY r.table_source, r.review_text