{{ config(materialized='table') }}

WITH review_dups AS (
    SELECT
        source_review_id,
        insert_id,
        hotel_id,
        COUNT(*) as dup_count,
        ARRAY_AGG(DISTINCT table_source) as sources
FROM {{ ref('all_reviews') }}
GROUP BY source_review_id, insert_id, hotel_id
HAVING COUNT(*) > 1
    )

SELECT
    'review' as entity_type,
    COALESCE(source_review_id, insert_id) as key,
    dup_count,
    sources
FROM review_dups
ORDER BY dup_count DESC