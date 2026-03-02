# Alpaca - Hotel Data Microservice

A comprehensive Go microservice suite that fetches, consolidates, and analyzes hotel data from multiple sources. 

Features multi-source data aggregation, RAG, and LLM-powered recommendation analysis.

Architected, coded, designed, and deployed by ChukiagoSoftware. 

Powered by Grok, ChatGPT, and Claude for code generation, code analysis, and research assistance.

Kubernetes, Cloud Native, with built-in metrics via OpenTelemetry.

## Architecture

Alpaca is a Go microservice suite. It does the following:
- Fetches hotel data from multiple sources
- Consolidates hotel data into a unified schema
- Fetches reviews from available API sources
- Implements Retrieval Augmented Generation - RAG -  with Google BigQuery and AI Search
- Implements LLM completion after a dimensional similarity search (GPT-4, Claude, Grok) to analyze reviews
- Generates intelligent recommendations based configurable LLM review analysis
- Stores data in SQLite (default) leveraging GORM for maintainability
- Uses a generalized provider interface for easy API integration
- Processes data in concurrent batches with rate limiting
- 

## Project Structure

```
alpaca/
├── alpaca/
│   ├── main.go                    # Main entry point - hotel data worker
│   ├── generate_cities.go         # City data generation utility (reference)
│   ├── generated_top_cities.go    # Generated top cities data (reference)
│   ├── REVIEW_PROCESSING.md       # Review processing documentation
│   ├── models/
│   │   ├── hotel.go              # Original Amadeus hotel models
│   │   └── hotel_extended.go     # Extended hotel models with recommendations
│   ├── services/
│   │   ├── hotel_service.go      # Hotel business logic (Amadeus)
│   │   ├── hotel_service_extended.go  # Extended hotel service (multi-source)
│   │   ├── review_crawler.go     # Review crawling from multiple sources
│   │   ├── llm_service.go        # LLM integration (GPT-4, Claude, Grok)
│   │   └── recommendation_service.go  # Recommendation orchestration
│   ├── database/
│   │   └── database.go           # SQLite database connection and schema
│   └── utils/
│       └── constants.go          # Constants and test data
├── go.mod                    # Go module definition
├── Dockerfile               # Docker build configuration
└── README.md                # This file
```

## Features

### ✅ Simplified Architecture
- **Single Microservice**: One focused service for hotel data collection
- **Raw SQL**: No ORM overhead, direct SQL control
- **SQLite First**: Simple, file-based database (easy to migrate to Postgres/Redshift later)
- **Generalized API Interface**: Easy to add new hotel data providers

### ✅ Multi-Source Hotel Data Collection
- **Amadeus API**: Hotel list, search, and ratings data
- **Expedia**: Hotel listings and reviews (interface ready)
- **Tripadvisor**: Hotel data and reviews (interface ready)
- **Google Places**: Hotel data and reviews (interface ready)
- **Booking.com**: Hotel data and reviews (interface ready)
- **Consolidated Schema**: Unified hotel table with ratings from all sources

### ✅ Review Processing & LLM Analysis
- **Multi-Source Review Crawling**: Automatically fetches reviews from:
  - Tripadvisor, Google, Expedia, Booking.com
  - Hotel websites, Bing, Yelp
- **LLM-Powered Analysis**: Uses GPT-4, Claude, or Grok to analyze reviews
- **Quality Detection**: Identifies hotels with excellent service, cleanliness, amenities
- **Quiet Detection**: Identifies quiet, peaceful hotels away from noise
- **Intelligent Recommendations**: Combines quality and quiet analysis for recommendations
- **Admin Override**: Admin flag to enable/disable hotels regardless of analysis

### ✅ Advanced Processing
- **Proper Pagination**: Handles multi-page API responses automatically
- **Concurrent Processing**: Uses goroutines for parallel data fetching
- **Rate Limiting**: Respects API limits with configurable delays
- **Error Handling**: Graceful degradation and detailed error logging
- **Invalid ID Tracking**: Skips hotel IDs that consistently fail

## 🚀 Getting Started

### Prerequisites
- Go 1.23+
- Amadeus API credentials (test or production)

### Environment Variables

Create a `.env` file in the project root:

```bash
# Amadeus API Credentials
AMD=your_client_id
AMS=your_client_secret

# Optional: Override default API URLs
AMADEUS_HOTEL_LIST_URL=https://test.api.amadeus.com/v1/reference-data/locations/hotels/by-city
AMADEUS_HOTEL_SEARCH_URL=https://test.api.amadeus.com/v2/shopping/hotel-offers
AMADEUS_HOTEL_RATINGS_URL=https://test.api.amadeus.com/v2/e-reputation/hotel-sentiments

# Optional: Database path (defaults to ./alpaca.db)
SQLITE_DB_PATH=./alpaca.db

# Optional: Search radius configuration
HOTEL_SEARCH_RADIUS=100
HOTEL_SEARCH_RADIUS_UNIT=MILE
```

### Running the Service

```bash
go build -o alpaca ./alpaca/alpaca
./alpaca
```

Or from the alpaca/alpaca directory:

```bash
cd alpaca/alpaca
go build -o alpaca
./alpaca
```

The service will:
1. Connect to SQLite database (creates if doesn't exist)
2. Fetch hotel list for Austin, TX (default city)
3. Fetch detailed search data for all hotels
4. Fetch ratings data for test hotel IDs

## Database Schema

The service uses a simple normalized schema with three main tables:

```sql
-- Basic hotel information
CREATE TABLE hotels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hotel_id TEXT UNIQUE NOT NULL,
    type TEXT,
    chain_code TEXT,
    dupe_id INTEGER,
    name TEXT,
    iata_code TEXT,
    address TEXT,        -- JSON stored as TEXT
    geo_code TEXT,      -- JSON stored as TEXT
    distance TEXT,      -- JSON stored as TEXT
    last_update TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Detailed hotel metadata
CREATE TABLE hotel_search_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hotel_id TEXT UNIQUE NOT NULL,
    type TEXT,
    chain_code TEXT,
    dupe_id INTEGER,
    name TEXT,
    rating INTEGER,
    official_rating INTEGER,
    description TEXT,   -- JSON stored as TEXT
    media TEXT,         -- JSON stored as TEXT
    amenities TEXT,     -- JSON stored as TEXT
    address TEXT,      -- JSON stored as TEXT
    contact TEXT,       -- JSON stored as TEXT
    policies TEXT,      -- JSON stored as TEXT
    available INTEGER DEFAULT 0,
    offers TEXT,        -- JSON stored as TEXT
    self TEXT,
    hotel_distance TEXT, -- JSON stored as TEXT
    last_update TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hotel_id) REFERENCES hotels(hotel_id)
);

-- Guest ratings and sentiment
CREATE TABLE hotel_ratings_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hotel_id TEXT UNIQUE NOT NULL,
    type TEXT,
    number_of_reviews INTEGER,
    number_of_ratings INTEGER,
    overall_rating INTEGER,
    sentiments TEXT,    -- JSON stored as TEXT
    last_update TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hotel_id) REFERENCES hotels(hotel_id)
);

-- Track invalid hotel IDs to skip in future runs
CREATE TABLE invalid_hotel_search_ids (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hotel_id TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## API Provider Interface

The service uses a generalized `HotelAPIProvider` interface, making it easy to add new hotel data sources:

```go
type HotelAPIProvider interface {
    GetOAuthToken(ctx context.Context) (string, error)
    FetchHotelsList(ctx context.Context, cityCode string, token string) ([]models.HotelAPIItem, string, error)
    FetchHotelSearchData(ctx context.Context, hotelID string, token string) (*models.HotelSearchData, error)
    FetchHotelRatingsData(ctx context.Context, hotelID string, token string) (*models.HotelRatingsData, error)
}
```

Currently implemented:
- `AmadeusProvider`: Full Amadeus API integration

Future providers can be added by implementing this interface.

## Data Flow

1. **OAuth Token**: Service authenticates with Amadeus API
2. **Hotel List**: Fetches basic hotel data by city (with pagination)
3. **Hotel IDs**: Extracts all hotel IDs for detailed processing
4. **Search Data**: Concurrently fetches detailed hotel metadata (5 concurrent requests)
5. **Ratings Data**: Concurrently fetches ratings and sentiment data (1 concurrent request for rate limiting)

## Performance Features

- **Concurrent Processing**: 5x faster data fetching with goroutines
- **Rate Limiting**: API-friendly request patterns (100-200ms delays)
- **Pagination Handling**: Efficient memory usage for large datasets
- **Database Indexing**: Optimized query performance
- **Error Recovery**: Graceful handling of API failures
- **Invalid ID Tracking**: Skips problematic hotel IDs automatically

## Review Processing

See [REVIEW_PROCESSING.md](alpaca/REVIEW_PROCESSING.md) for detailed documentation on:
- Review crawling from multiple sources
- LLM analysis for Quality and Quiet detection
- Recommendation generation
- Usage examples

### Quick Start - Review Processing

```go
// Initialize services
db, _ := database.NewDatabase()
hotelService := services.NewhotelService(db)
reviewCrawler := services.NewReviewCrawlerService(db)
llmProvider := services.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY"))
llmService := services.NewLLMService(llmProvider)
recommendationService := services.NewRecommendationService(
    hotelService, reviewCrawler, llmService,
)

// Process recommendations for a hotel
err := recommendationService.ProcessHotelRecommendations(ctx, "hotel-id")
```

## Next Steps & Recommendations

### Database Backend Options

1. **SQLite (Current)**: Good for development and small datasets
   - Pros: Simple, no server needed, fast for reads
   - Cons: Limited concurrency, not ideal for high write loads

2. **PostgreSQL**: Recommended for production
   - Pros: Better concurrency, JSON support, full SQL features
   - Cons: Requires server setup

3. **AWS Redshift**: For analytics workloads
   - Pros: Columnar storage, optimized for analytics
   - Cons: More complex setup, better for read-heavy analytics

4. **MongoDB**: If you need document flexibility
   - Pros: Native JSON, flexible schema
   - Cons: Different query model, may need to rethink relationships

**Recommendation**: Start with SQLite for development, migrate to PostgreSQL for production. The raw SQL approach makes migration straightforward.

### Code Simplification Opportunities

1. **Struct Simplification**: 
   - Consider flattening some nested JSON structures
   - Remove unused fields from API responses
   - Create separate structs for database vs API models

2. **Database Code**:
   - Add connection pooling configuration
   - Implement prepared statements for better performance
   - Add transaction support for batch operations

3. **Error Handling**:
   - Create custom error types for better error handling
   - Add retry logic with exponential backoff
   - Implement circuit breaker pattern for API calls

4. **Configuration**:
   - Move hardcoded values to config file
   - Add validation for environment variables
   - Support multiple city codes

5. **Testing**:
   - Add unit tests for database operations
   - Add integration tests for API provider
   - Mock external API calls for testing

## Development

### Building

```bash
go build -o alpaca ./alpaca/alpaca
```

### Running Tests

```bash
go test ./...
```

### Code Style

The project follows standard Go conventions:
- Use `gofmt` for formatting
- Use `golint` for linting
- Follow Go naming conventions

## License

MIT
