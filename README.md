# Alpaca

A Go HTTP frontend with Postgres, SQLite or other cloud databases. API to query Hotel data. Further will use analytics for sentiment analysis and GPT to query accommodations based on additional criteria or user input.

Based on Platzi's Advanced Go Course and built using Cursor AI, OpenAI ChatGPT and Twitter's Grok. Most of the code at this point is AI generated and improved upon by prompt. Testing is also automatic and further tests will be added.

## Exploration

LLM choice and library or framework. Retrieval Augmented Generation (RAG) fine-tuning for this and other datasets.


## 🏨 Hotel Data System

### API OVERVIEW
Alpaca now includes a comprehensive hotel data management system that fetches and stores detailed hotel information from multiple Amadeus APIs:

- **Hotel List API**: Basic hotel information by city
- **Hotel Search API**: Detailed hotel metadata, amenities, and offers
- **Hotel Ratings API**: Sentiment analysis and guest ratings

### Architecture

#### Database Models with Foreign Key Relationships
- `HotelAPIItem`: Basic hotel information (name, location, chain code) - **Parent Table**
- `HotelSearchData`: Detailed hotel metadata (amenities, media, policies, offers) - **Child Table**
- `HotelRatingsData`: Guest ratings and sentiment analysis (overall rating, service quality, etc.) - **Child Table**

**Foreign Key Relationships:**
- `HotelSearchData.hotel_id` → `HotelAPIItem.hotel_id` (1:1 relationship)
- `HotelRatingsData.hotel_id` → `HotelAPIItem.hotel_id` (1:1 relationship)

#### Data Flow
```
Hotel List API → Basic hotel data (hotel_api_items)
     ↓
Hotel Search API → Detailed metadata (hotel_search_data)  
     ↓
Hotel Ratings API → Sentiment data (hotel_ratings_data)
```

### Features

#### ✅ Multi-API Integration
- **Amadeus Hotel List API**: Fetches hotels by city with pagination
- **Amadeus Hotel Search API**: Retrieves detailed hotel information
- **Amadeus Hotel Ratings API**: Gets guest sentiment and ratings

#### ✅ Advanced Data Fetching
- **Proper Pagination**: Handles multi-page API responses automatically
- **Concurrent Processing**: Uses goroutines for parallel data fetching
- **Rate Limiting**: Respects API limits with configurable delays
- **Error Handling**: Graceful degradation and detailed error logging

#### ✅ Database Flexibility
- **Multi-Database Support**: SQLite (default) and PostgreSQL
- **Environment Configuration**: Easy switching via `DATABASE_TYPE`
- **Shared Database**: Worker and main app use the same database file
- **GORM Integration**: Modern ORM with automatic migrations
- **Foreign Key Relationships**: Proper referential integrity with GORM

#### ✅ Service Layer
- **HotelService**: Complete CRUD operations for all hotel data types
- **Concurrent Safe**: Thread-safe operations with proper locking
- **Context Support**: Full context propagation for timeouts and cancellation
- **Relationship Support**: Preload related data with GORM associations
- **Upsert Operations**: Smart create/update operations for data consistency

## 🚀 Getting Started

### Running the Applications

#### Main Application
```bash
go build -o alpaca
./alpaca
```

#### Hotel Data Worker
```bash
cd worker-alpaca
go build -o worker-alpaca
./worker-alpaca
```

### API Endpoints

#### Hotel Endpoints (Public)
- `GET /api/v1/hotels` - List all hotels
- `GET /api/v1/hotels?withDetails=true` - List hotels with search and ratings data
- `GET /api/v1/hotels/{hotelId}` - Get specific hotel
- `GET /api/v1/hotels/{hotelId}?withDetails=true` - Get hotel with search and ratings data
- `GET /api/v1/hotels/city/{cityName}` - Get hotels by city
- `GET /api/v1/hotels/city/{cityName}?withDetails=true` - Get hotels by city with details

#### Advanced Hotel Endpoints
- `GET /api/v1/hotels/complete` - Get hotels with complete data (all three types)
- `GET /api/v1/hotels/with-search` - Get hotels that have search data
- `GET /api/v1/hotels/with-ratings` - Get hotels that have ratings data

## 🔧 Technical Implementation

### Worker Architecture
The worker implements a three-phase data collection strategy:

1. **Phase 1**: Fetch basic hotel list with pagination
2. **Phase 2**: Extract hotel IDs for detailed processing
3. **Phase 3**: Concurrently fetch search data (5 concurrent requests)
4. **Phase 4**: Concurrently fetch ratings data (5 concurrent requests)

### Concurrency Features
- **Semaphore Limiting**: Maximum 5 concurrent API requests
- **WaitGroup Synchronization**: Proper goroutine coordination
- **Mutex Protection**: Thread-safe counters and shared state
- **Rate Limiting**: 100ms for list API, 200ms for search/ratings APIs

### Database Schema with Foreign Keys
```sql
-- Basic hotel information (Parent Table)
hotel_api_items (
  id, created_at, updated_at, deleted_at,
  type, hotel_id (UNIQUE), chain_code, dupe_id, name, iata_code,
  address (JSON), geo_code (JSON), distance (JSON), last_update
)

-- Detailed hotel metadata (Child Table)
hotel_search_data (
  id, created_at, updated_at, deleted_at,
  type, hotel_id (UNIQUE, FOREIGN KEY), chain_code, dupe_id, name, rating, official_rating,
  description (JSON), media (JSON), amenities (JSON), address (JSON),
  contact (JSON), policies (JSON), available, offers (JSON), self,
  hotel_distance (JSON), last_update
)

-- Guest ratings and sentiment (Child Table)
hotel_ratings_data (
  id, created_at, updated_at, deleted_at,
  type, hotel_id (UNIQUE, FOREIGN KEY), number_of_reviews, number_of_ratings, overall_rating,
  sentiments (JSON), last_update
)
```

### GORM Relationships
```go
// Parent table with relationships
type HotelAPIItem struct {
    // ... fields ...
    SearchData  *HotelSearchData  `gorm:"foreignKey:HotelID;references:HotelID"`
    RatingsData *HotelRatingsData `gorm:"foreignKey:HotelID;references:HotelID"`
}

// Child tables with back-references
type HotelSearchData struct {
    // ... fields ...
    Hotel *HotelAPIItem `gorm:"foreignKey:HotelID;references:HotelID"`
}

type HotelRatingsData struct {
    // ... fields ...
    Hotel *HotelAPIItem `gorm:"foreignKey:HotelID;references:HotelID"`
}
```

## 🏗️ Project Structure

```
alpaca/
├── main.go                 # Main application entry point
├── worker-alpaca/
│   └── main.go            # Hotel data worker
├── models/
│   └── hotel.go           # Hotel data models with GORM relationships
├── services/
│   └── hotel_service.go   # Hotel business logic with relationship support
├── handlers/
│   └── hotel.go           # HTTP handlers with detailed data endpoints
├── database/
│   └── database.go        # Database factory
└── server/
    └── server.go          # Server configuration
```

## 🔄 Data Flow

1. **Worker fetches hotel list** from Amadeus by city (Austin, TX)
2. **Basic hotel data** is stored in `hotel_api_items` table
3. **Worker extracts hotel IDs** for detailed processing
4. **Concurrent goroutines** fetch search and ratings data
5. **Detailed data** is stored in respective tables with foreign key constraints
6. **Main application** serves data via REST API endpoints with relationship support

## 🎯 Future Enhancements

- **STAR Schema**: Analytical data warehouse for complex queries
- **Real-time Updates**: WebSocket notifications for data changes
- **Caching Layer**: Redis integration for performance
- **Analytics Dashboard**: Hotel performance metrics
- **Multi-city Support**: Expand beyond Austin to other cities

## 📊 Performance Features

- **Concurrent Processing**: 5x faster data fetching
- **Rate Limiting**: API-friendly request patterns
- **Pagination Handling**: Efficient memory usage
- **Database Indexing**: Optimized query performance with foreign key indexes
- **Error Recovery**: Graceful handling of API failures
- **Relationship Queries**: Efficient joins using GORM preloading

## 🔒 Security

- **Environment Variables**: Secure credential management
- **API Rate Limiting**: Prevents API abuse
- **Input Validation**: Sanitized API parameters
- **Database Isolation**: Separate user and hotel data
- **Referential Integrity**: Foreign key constraints prevent orphaned records

## 🚀 Planned Analytics & AI Integration

### STAR Schema Architecture
The hotel data will be transformed into a STAR schema for advanced analytics:

- **Fact Tables**: Hotel metrics, ratings, availability trends
- **Dimension Tables**: Hotels, locations, amenities, time periods
- **Analytics Layer**: Aggregated KPIs and business intelligence

### Go AI Service
A dedicated Go service will handle LLM inference for hotel analytics:

- **Library**: `github.com/sashabaranov/go-openai` for OpenAI API integration
- **Primary LLM**: GPT-4 for advanced hotel analysis and recommendations
- **Alternative LLMs**: Claude (Anthropic) and Grok (X) for comparison
- **Use Cases**: 
  - Hotel recommendation engine
  - Sentiment analysis insights
  - Pricing optimization suggestions
  - Guest experience predictions

### Implementation Plan
1. **STAR Schema Migration**: Transform current normalized data
2. **Analytics Service**: Go service with REST API endpoints
3. **LLM Integration**: OpenAI GPT-4 API with fallback options
4. **Real-time Inference**: WebSocket connections for live recommendations

---

## Legacy Features

### Current Implemented
- Code from Platzi Advanced Go and Websockets Go courses
- Websockets Hub for single direction messaging
- HTTP Gorilla Mux server
- Docker support

### In Progress
- Docker build and optimization
- Advanced analytics and reporting

### Deployed
- Deployed on Oracle Cloud server using Cloud Init with Docker