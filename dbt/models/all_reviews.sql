{{ config(materialized='view') }}

{{ dbt_utils.union_relations(
    relations = [
        source('bq', 'main_hotel_reviews_all_110k'),
        source('bq', 'bigReviews'),
        source('bq', 'reviews'),
        source('bq', 'hotel_reviews_uploaded_to_BQ'),
    ],
    source_column_name = 'table_source'
) }}
