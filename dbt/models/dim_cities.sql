{{ config(materialized='table') }}

SELECT
    name as city,
    country,
    continent,
    'bigCity' as city_source
FROM {{ source('bq', 'bigCity') }}

UNION ALL

SELECT
    name as city,
    country,
    NULL as continent,
    'cities' as city_source
FROM {{ source('bq', 'cities') }}

UNION ALL

SELECT
    name as city,
    country,
    NULL as continent,
    'airportCity' as city_source
FROM {{ source('bq', 'airportCities') }}