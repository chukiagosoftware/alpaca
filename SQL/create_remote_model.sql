CREATE
OR
REPLACE MODEL
  `golang1212025.alpacaCentral.alpacaremote` -- Replace 'my_remote_embedding_model' with your desired model name
REMOTE
WITH CONNECTION `golang1212025.us-central1.alpaca` -- Replace 'your_connection_id' with the actual ID of your connection
    OPTIONS (
    ENDPOINT = 'text-embedding-004'
    );
