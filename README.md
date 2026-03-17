# Alpaca - Hotel Review Search turbocharged with AI

Find Hotel Reviews with fine-tuned AI powered search criteria. Comprehensive Go microservice suite to fetch, consolidate, and analyze hotel review data with RAG and LLM.

Uses Google Vertex AI Vector Search for Retrieval Augmentd Generation RAG, and LLM-powered recommendation analysis.

Architected, coded, designed, and deployed by ChukiagoSoftware. 

Development enhanced by Grok, ChatGPT, and Claude for code generation, code analysis, and research assistance.

Kubernetes, Docker and GCP cloud native, with built-in metrics via OpenTelemetry.

## Architecture

Alpaca is a Go microservice suite for LLMs and RAG with SQL and BigQuery data. 

It does the following:
- Fetches hotel and review data from multiple sources
- Consolidates data into a unified schema
- Generates vector Embeddings using Gemini-003 or text-embedding-004 from Google
- Implements Retrieval Augmented Generation - RAG -  with Google BigQuery and Vertex AI Vector Search
- Implements Gemini LLM completion after a dimensional similarity search (supports GPT-4, Claude, Grok)
- Generates intelligent recommendations for a user question based on configurable LLM review analysis
- Stores development data in SQLite (default) leveraging GORM for maintainability
- Uses a generalized provider interface for easy API integration
- Processes data in concurrent batches with rate limiting
- Scrape complex CDN websites for hotel reviews using AppleScript, Go Colly.



### ✅ Multi-Source Hotel Data Collection
- **Amadeus API**: Curated worldwide hotel list, guest sentiments and ratings
- **Expedia**: Hotel listings and user reviews (interface ready)
- **Tripadvisor**: Hotel data and user reviews (interface ready)
- **Google Places**: Hotel data and user reviews 
- **Booking.com**: Hotel data and reviews (interface ready)
- **Consolidated Schema**: Unified hotel table with ratings from all sources

### Modular design can support multiple data sources, providers, LLMs, Vector Database/Search services

### Go Gin HTTP server, Alpine Docker image build optimized for rapid cloud deployment

### Integrated OpenTelemetry tracing for LLM and RAG operations
