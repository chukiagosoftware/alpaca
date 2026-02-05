# Review Processing and Recommendation System

## Overview

The system now supports:
1. **Multi-source hotel data** - Consolidates hotels from Amadeus, Expedia, Tripadvisor, Google, Booking.com
2. **Review crawling** - Automatically fetches reviews from multiple sources
3. **LLM analysis** - Uses GPT-4, Claude, or Grok to analyze reviews for Quality and Quiet
4. **Recommendation engine** - Generates recommendations based on LLM analysis

## Database Schema

### Hotels Table (Consolidated)
- **Basic Info**: name, city, country, address, coordinates
- **Ratings**: Separate ratings from each source (amadeus_rating, expedia_rating, etc.)
- **Recommendation Fields**:
  - `recommended` (boolean): Overall recommendation
  - `admin_flag` (boolean): Admin override (true = disabled)
  - `quality` (boolean): Has quality based on reviews
  - `quiet` (boolean): Is quiet based on reviews
  - `important_note` (text): Notes about recommendation calculation

### Hotel Reviews Table
- Stores reviews from all sources
- Fields: source, reviewer_name, rating, review_text, review_date, verified, etc.
- Unique constraint on (hotel_id, source, source_review_id)

### Hotel Recommendations Table
- LLM-processed recommendations
- Quality and Quiet scores (0.0 to 1.0) with confidence levels
- Reasoning text explaining the analysis
- Overall recommendation boolean

## Review Sources

The system crawls reviews from:
1. **Tripadvisor** - Largest review platform
2. **Google** - Google Places reviews
3. **Expedia** - Booking site reviews
4. **Booking.com** - Major booking platform
5. **Hotel Website** - Direct from hotel's own site
6. **Bing** - Search engine reviews
7. **Yelp** - Local business reviews

## LLM Analysis

### Quality Analysis
**Positive Indicators:**
- Excellent service, staff helpfulness
- Cleanliness, well-maintained
- Good amenities, facilities
- Comfortable rooms, good beds
- Good value for money
- Positive guest experiences

**Negative Indicators:**
- Poor service, unhelpful staff
- Dirty, unclean conditions
- Broken amenities, poor maintenance
- Uncomfortable rooms
- Poor value, overpriced
- Safety concerns

### Quiet Analysis
**Quiet Indicators (POSITIVE):**
- "quiet", "peaceful", "tranquil", "serene"
- "not on any major roads", "away from traffic"
- "cul de sac", "residential area", "quiet neighborhood"
- "soundproof", "well-insulated", "thick walls"
- "no street noise", "no traffic noise"
- "peaceful location", "quiet area"
- "good sleep", "restful", "relaxing atmosphere"
- "away from city center noise"
- "garden view", "courtyard", "interior room"
- "no construction nearby"

**Noise Indicators (NEGATIVE - Disqualifiers):**
- "noisy", "loud", "can't sleep"
- "thin walls", "hear neighbors", "sound travels"
- "construction", "renovation", "building work"
- "street noise", "traffic noise", "road noise"
- "airport noise", "airplane noise"
- "nightclub nearby", "bar nearby", "music"
- "elevator noise", "hallway noise"
- "party", "rowdy guests"
- "main road", "highway", "busy street"
- "train noise", "railway"
- "thin windows", "poor insulation"

## Usage

### Processing Recommendations for a Single Hotel

```go
// Initialize services
db, _ := database.NewDatabase()
hotelService := services.NewHotelService(db)
reviewCrawler := services.NewReviewCrawlerService(db)
llmProvider := services.NewOpenAIProvider("your-api-key")
llmService := services.NewLLMService(llmProvider)
recommendationService := services.NewRecommendationService(
    hotelService,
    reviewCrawler,
    llmService,
)

// Process a single hotel
err := recommendationService.ProcessHotelRecommendations(ctx, "hotel-id-123")
```

### Processing All Hotels

```go
err := recommendationService.ProcessAllHotels(ctx)
```

### Updating Admin Flag

```go
// Disable a hotel (overrides all recommendations)
hotelService.UpdateAdminFlag(ctx, "hotel-id-123", true)

// Re-enable a hotel
hotelService.UpdateAdminFlag(ctx, "hotel-id-123", false)
```

## LLM Providers

### OpenAI (GPT-4)
```go
provider := services.NewOpenAIProvider("your-openai-api-key")
// Uses GPT-4 by default, set OPENAI_MODEL env var to change
```

### Claude (Anthropic)
```go
provider := services.NewClaudeProvider("your-anthropic-api-key")
// Uses claude-3-opus-20240229 by default
```

### Grok (X/Twitter)
```go
provider := services.NewGrokProvider("your-grok-api-key")
// Uses grok-beta by default
```

## Environment Variables

```bash
# LLM API Keys
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4  # Optional, defaults to gpt-4
ANTHROPIC_API_KEY=sk-ant-...
CLAUDE_MODEL=claude-3-opus-20240229  # Optional
GROK_API_KEY=...
GROK_MODEL=grok-beta  # Optional

# Database
SQLITE_DB_PATH=./alpaca.db
```

## Recommendation Logic

A hotel is **recommended** if:
1. `admin_flag` is false (not disabled by admin)
2. `quality` is true (quality score >= 0.7)
3. `quiet` is true (quiet score >= 0.7)
4. Overall recommendation from LLM is true

The `admin_flag` overrides everything - if true, the hotel is never recommended regardless of other factors.

## Next Steps

1. **Implement Review Crawlers**: The crawler interfaces are defined but need implementation for each source
2. **Add Rate Limiting**: Implement proper rate limiting for API calls
3. **Batch Processing**: Optimize LLM calls to process multiple hotels efficiently
4. **Caching**: Cache LLM responses to avoid re-processing unchanged reviews
5. **Monitoring**: Add metrics and logging for production use
