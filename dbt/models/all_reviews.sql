{{ config(materialized='view') }}

WITH unioned AS (SELECT id,
                        hotel_id,
                        source_review_id,
                        insert_id,
                        review_text,
                        reviewer_name,
                        reviewer_location,
                        rating,
                        review_date,
                        google_maps_uri,
                        photo_name,
                        'main_hotel_reviews_all_110k' as table_source
FROM {{ source ('bq', 'main_hotel_reviews_all_110k') }}

UNION ALL
SELECT id,
    hotel_id,
    NULL      as source_review_id,
    NULL      as insert_id,
    review_text,
    reviewer_name,
    NULL      as reviewer_location,
    NULL      as rating,
    NULL      as review_date,
    NULL      as google_maps_uri,
    NULL      as photo_name,
    'reviews' as table_source

       'bigReview' as table_source
FROM {{ source('bq', 'bigReviews') }}

UNION ALL
SELECT id,
    hotel_id,
    NULL      as source_review_id,
    NULL      as insert_id,
    review_text,
    reviewer_name,
    NULL      as reviewer_location,
    NULL      as rating,
    NULL      as review_date,
    NULL      as google_maps_uri,
    NULL      as photo_name,
    'reviews' as table_source
FROM {{ source('bq', 'reviews') }}

UNION ALL
SELECT *, 'intermediate_upload' as table_source
FROM {{ source('bq', 'hotel_reviews_uploaded_to_BQ') }}
),

with_hash AS (
SELECT
    *, MD5(TRIM(REGEXP_REPLACE(LOWER(review_text), r'\s+', ' '))) as review_text_hash
FROM unioned
WHERE review_text IS NOT NULL
    )

SELECT id,
       hotel_id,
       source_review_id,
       insert_id,
       review_text,
       reviewer_name,
       reviewer_location,
       rating,
       review_date,
       google_maps_uri,
       photo_name,
       table_source,
       review_text_hash
FROM with_hash
QUALIFY ROW_NUMBER() OVER (PARTITION BY review_text_hash ORDER BY table_source DESC, id DESC) = 1