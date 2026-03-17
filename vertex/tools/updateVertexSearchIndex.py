import sqlite3
import pandas as pd
from google.cloud import bigquery
from google.cloud import aiplatform
from google.cloud.aiplatform.matching_engine import MatchingEngineIndex
import time
import json
import os

# --- Configuration ---
PROJECT_ID = 'golang1212025' # Your Google Cloud Project ID
REGION = 'us-central1' # Region where your Vector Search index is
BIGQUERY_DATASET = 'alpacaCentral'
BIGQUERY_TABLE = 'review_embeddings_saopaolo_helsinki' # New table for processed data
LOCAL_SQLITE_DB = '../alpaca.db' # Your local SQLite database file
VECTOR_SEARCH_INDEX_ID = '7927912043846303744' # Your Vector Search Index ID
GCS_BUCKET_NAME = f'{PROJECT_ID}-vector-search-imports' # GCS bucket for temporary files

# Initialize BigQuery client
bq_client = bigquery.Client(project=PROJECT_ID)

# Initialize AI Platform client (for Gemini and Vector Search)
aiplatform.init(project=PROJECT_ID, location=REGION)

# --- 1. Export from local sqlite3 hotel_reviews table ---
def export_from_sqlite():
    print(f"1. Exporting data from local SQLite database: {LOCAL_SQLITE_DB}")
    try:
        conn = sqlite3.connect(LOCAL_SQLITE_DB)
        query = """
        SELECT hotel_id, review_text, reviewer_name, rating, city, country
        FROM hotel_reviews
        """
        df = pd.read_sql_query(query, conn)
        conn.close()
        print(f"   Exported {len(df)} rows from SQLite.")
        return df
    except Exception as e:
        print(f"Error exporting from SQLite: {e}")
        return pd.DataFrame() # Return empty DataFrame on error

# --- 2. Export that to BigQuery alpacacentral dataset, review_embeddings_saopaolo_helsinki table ---
def upload_to_bigquery(df):
    if df.empty:
        print("   No data to upload to BigQuery.")
        return

    print(f"2. Uploading data to BigQuery table: {BIGQUERY_DATASET}.{BIGQUERY_TABLE}")
    table_id = f"{PROJECT_ID}.{BIGQUERY_DATASET}.{BIGQUERY_TABLE}"

    # Define BigQuery schema for the raw data
    schema = [
        bigquery.SchemaField("hotel_id", "INTEGER"),
        bigquery.SchemaField("review_text", "STRING"),
        bigquery.SchemaField("reviewer_name", "STRING"),
        bigquery.SchemaField("rating", "INTEGER"),
        bigquery.SchemaField("city", "STRING"),
        bigquery.SchemaField("country", "STRING"),
    ]

    job_config = bigquery.LoadJobConfig(
        schema=schema,
        write_disposition=bigquery.WriteDisposition.WRITE_TRUNCATE, # Overwrite the table
    )

    try:
        job = bq_client.load_table_from_dataframe(
            df, table_id, job_config=job_config
        )
        job.result() # Wait for the job to complete
        print(f"   Loaded {job.output_rows} rows into BigQuery table: {table_id}")
    except Exception as e:
        print(f"Error uploading to BigQuery: {e}")

# --- 3. Generate gemini-001 text embeddings and save that in the new table overwriting it ---
def generate_and_save_embeddings():
    print("3. Generating Gemini-001 text embeddings and saving to BigQuery.")
    bq_table_path = f"{PROJECT_ID}.{BIGQUERY_DATASET}.{BIGQUERY_TABLE}"

    # Use BigQuery ML to generate embeddings
    # This query will fetch review_text, generate embeddings, and store them along with original fields
    embedding_query = f"""
    CREATE OR REPLACE TABLE `{bq_table_path}` AS
    SELECT
        t.hotel_id,
        t.review_text,
        t.reviewer_name,
        t.rating,
        t.city,
        t.country,
        ml_generate_embedding_result.embeddings AS review_embedding
    FROM
        `{bq_table_path}` AS t,
        ML.GENERATE_TEXT_EMBEDDING(
            MODEL `gcp-public-models.llm.gemini-001`,
            TABLE `{bq_table_path}`,
            STRUCT(
                'review_text' AS text_column,
                FALSE AS flatten_json_output
            )
        ) AS ml_generate_embedding_result
    """
    try:
        query_job = bq_client.query(embedding_query)
        query_job.result() # Wait for the job to complete
        print(f"   Embeddings generated and saved to BigQuery table: {bq_table_path}")
    except Exception as e:
        print(f"Error generating embeddings with BigQuery ML: {e}")

# --- 4. Run the ImportIndex to my index, upserting the new embeddings ---
def import_embeddings_to_vector_search():
    print("4. Preparing and importing embeddings to Vector Search index.")
    bq_table_path = f"{PROJECT_ID}.{BIGQUERY_DATASET}.{BIGQUERY_TABLE}"
    gcs_output_folder = f"gs://{GCS_BUCKET_NAME}/vector_search_imports/{int(time.time())}/"

    # Create the GCS bucket if it doesn't exist
    print(f"   Ensuring GCS bucket {GCS_BUCKET_NAME} exists...")
    try:
        # Using gsutil via os.system for simplicity, but google-cloud-storage library is more robust
        os.system(f'gsutil mb -p {PROJECT_ID} gs://{GCS_BUCKET_NAME}')
        print(f"   GCS bucket {GCS_BUCKET_NAME} checked/created.")
    except Exception as e:
        print(f"Warning: Could not create GCS bucket (might already exist or permission issue): {e}")


    # Export embeddings from BigQuery to GCS in JSONL format
    # The output format for Vector Search needs "id" and "embedding" fields.
    # We will use hotel_id as the Vector Search ID.
    export_query = f"""
    EXPORT DATA OPTIONS(
        uri='{gcs_output_folder}embeddings_*.jsonl',
        format='NEWLINE_DELIMITED_JSON',
        overwrite=true
    ) AS
    SELECT
        CAST(hotel_id AS STRING) AS id, -- Vector Search expects ID as string
        review_embedding AS embedding,
        -- Optionally, you can add other fields as metadata for filtering,
        -- but ensure your index configuration supports them.
        TO_JSON(STRUCT(reviewer_name, rating, city, country)) AS rest_metadata
    FROM
        `{bq_table_path}`
    WHERE
        ARRAY_LENGTH(review_embedding) > 0 -- Ensure only rows with valid embeddings are exported
    """

    print(f"   Exporting embeddings from BigQuery to GCS: {gcs_output_folder}")
    try:
        query_job = bq_client.query(export_query)
        query_job.result()
        print(f"   Embeddings successfully exported to GCS.")
    except Exception as e:
        print(f"Error exporting embeddings to GCS: {e}")
        return

    # Now, initiate the ImportIndex operation
    index = aiplatform.MatchingEngineIndex(index_name=VECTOR_SEARCH_INDEX_ID)

    try:
        # `update_embeddings` performs an upsert operation
        print(f"   Initiating batch update (upsert) to Vector Search index: {VECTOR_SEARCH_INDEX_ID}")
        # The `contents_delta_uri` points to the GCS folder containing the JSONL files.
        operation = index.update_embeddings(contents_delta_uri=gcs_output_folder)
        print(f"   Vector Search update operation started: {operation.operation.name}")
        operation.wait() # Wait for the operation to complete
        print("   Vector Search update operation completed successfully.")
        print(f"   New vector count should reflect the imported data. Please check in the console.")
    except Exception as e:
        print(f"Error during Vector Search update: {e}")

# --- Main execution ---
if __name__ == "__main__":
    # Ensure a dummy SQLite DB and table exist for testing if you don't have one
    if not os.path.exists(LOCAL_SQLITE_DB):
        conn = sqlite3.connect(LOCAL_SQLITE_DB)
        cursor = conn.cursor()
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS hotel_reviews (
                hotel_id INTEGER PRIMARY KEY,
                review_text TEXT,
                reviewer_name TEXT,
                rating INTEGER,
                city TEXT,
                country TEXT
            );
        """)
        cursor.execute("INSERT INTO hotel_reviews VALUES (1, 'Great stay in Saopaolo!', 'Alice', 5, 'Saopaolo', 'Brazil');")
        cursor.execute("INSERT INTO hotel_reviews VALUES (2, 'Lovely hotel in Helsinki.', 'Bob', 4, 'Helsinki', 'Finland');")
        cursor.execute("INSERT INTO hotel_reviews VALUES (3, 'Not bad, in Saopaolo.', 'Charlie', 3, 'Saopaolo', 'Brazil');")
        conn.commit()
        conn.close()
        print(f"Created dummy SQLite DB: {LOCAL_SQLITE_DB}")

    # 1. Export from local SQLite
    reviews_df = export_from_sqlite()

    if not reviews_df.empty:
        # 2. Upload to BigQuery
        upload_to_bigquery(reviews_df)

        # 3. Generate embeddings in BigQuery
        generate_and_save_embeddings()

        # 4. Import to Vector Search
        import_embeddings_to_vector_search()
    else:
        print("No data processed due to empty DataFrame.")

