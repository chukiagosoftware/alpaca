# Alpaca - Hotel Review Search turbocharged with AI

Find your next hotel based on relevant and recent human reviews. Comprehensive Go microservice suite to fetch and consolidate review data with RAG and LLM for post-processing.

Serving with Go Gin HTTP server and built-in AI traces/metrics with OpenTelemetry.

Performant backend uses Google Vertex AI Vector Search for Retrieval Augmented Generation (RAG), and Gemini LLM-powered recommendation analysis. Both are modular and will be tested against AWS Bedrock, S3 Vectors and open source vector databases (Qdrant).

Architected, coded, designed, and deployed by ChukiagoSoftware. 

Developed leveraging Grok and Claude for code generation, research assistance and improved by ChukiagoSoftware's own expertise.

## System Overview

Alpaca is a Go microservice suite for LLMs and RAG with SQL and BigQuery, AWS Glue, S3, Bedrock and multiple other providers as modular options. 

It **does** the following:

### Features
- Fetch consolidated hotel and review data from multiple public API including Google Places, Tripadvisor, Amadeus.
- Generate vector Embeddings using Gemini-003 or text-embedding-004 from Google, or modular options including AWS Bedrock, S3 Vectors, or Python low-level PyTorch.
- Implements Retrieval Augmented Generation with Google BigQuery and Vertex AI Vector Index Search.
- Can easily be ported to AWS or Qdrant.
- Implements LLM completion after a dimensional similarity search (Gemini, GPT-4, Claude, Grok via standard Go `net/http` or provider SDK.
- Generates intelligent recommendations for a user question based on configurable LLM prompts and custom optimizations.


### Software engineering
- Store development data in SQLite (default) leveraging GORM for maintainability
- Uses a generalized provider interface for easy API integration
- Modular design with a performant backend ready to serve React/Vue/Angular or standard HTML/JS.
- Process data in concurrent batches with rate limiting
- Separate concerns for data fetching, processing, embedding, storage, and retrieval.
- Scrape complex CDN websites for hotel reviews using AppleScript, Go Colly.
- Leverage available provider APIs for data collection and analysis.

### Infrastructure
- Deployed on Google Cloud using Pulumi Go
- Optimized two-stage Docker build for production with lightweight images.
- Runs native on Cloud Run, Kubernetes, or other services using standard Docker image and registry.

