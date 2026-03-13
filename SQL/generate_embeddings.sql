-- Todo switch to AI.Generate_Embedding to avoid IAM and BQ service permissions

CREATE TABLE ` golang1212025.alpacaCentral.review_embeddings `
AS

SELECT t.id, t.review_text, embeddings.ml_generate_embedding_result
FROM `golang1212025.alpacaCentral.reviews` AS t
         JOIN
     ML.GENERATE_EMBEDDING(
             MODEL `golang1212025.alpacaCentral.alpacaremote`,
             (SELECT id, review_text AS content
              FROM `golang1212025.alpacaCentral.reviews`
              WHERE review_text IS NOT NULL
                AND LENGTH(review_text) > 0),
             STRUCT (TRUE AS flatten_json_output))
         AS embeddings
     ON t.id = embeddings.id;




