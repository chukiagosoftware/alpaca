CREATE OR REPLACE TABLE `golang1212025.alpacaCentral.review_embeddings`
AS
SELECT
    r.id,
    r.review_text,
    h.name AS hotel_name, -- Assuming 'name' is the hotel's name column
    h.City AS city,
    h.Country AS country,
    embeddings.ml_generate_embedding_result AS embedding
FROM
    `golang1212025.alpacaCentral.reviews` AS r
        JOIN
    `golang1212025.alpacaCentral.hotels` AS h
    ON
        r.hotel_id = h.hotel_id -- Corrected join condition
        JOIN
    ML.GENERATE_EMBEDDING(
            MODEL `golang1212025.alpacaCentral.alpaca-gemini-embedding-001`,
            (
                SELECT reviews.id, reviews.review_text AS content
                FROM `golang1212025.alpacaCentral.reviews` AS reviews
                WHERE reviews.review_text IS NOT NULL AND LENGTH(reviews.review_text) > 0
            ),
            STRUCT(TRUE AS flatten_json_output)
    ) AS embeddings
    ON
        r.id = embeddings.id
WHERE r.review_text IS NOT NULL AND LENGTH(r.review_text) > 0;
