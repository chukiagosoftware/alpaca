CREATE OR REPLACE TABLE `golang1212025.alpacaCentral.review_embeddings` AS

SELECT
    * EXCEPT(row_num)
FROM (
         SELECT
             *,
             ROW_NUMBER() OVER (PARTITION BY hotel_name, review_text ORDER BY hotel_name) AS row_num
         FROM
             `golang1212025.alpacaCentral.review_embeddings`
     )
WHERE
    row_num = 1
