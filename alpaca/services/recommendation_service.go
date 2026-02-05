package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/edamsoft-sre/alpaca/alpaca/models"
)

// RecommendationService orchestrates the review crawling and LLM analysis
type RecommendationService struct {
	hotelService  *HotelService
	reviewCrawler *ReviewCrawlerService
	llmService    *LLMService
}

// NewRecommendationService creates a new recommendation service
func NewRecommendationService(
	hotelService *HotelService,
	reviewCrawler *ReviewCrawlerService,
	llmService *LLMService,
) *RecommendationService {
	return &RecommendationService{
		hotelService:  hotelService,
		reviewCrawler: reviewCrawler,
		llmService:    llmService,
	}
}

// ProcessHotelRecommendations processes reviews and generates recommendations for a hotel
func (s *RecommendationService) ProcessHotelRecommendations(ctx context.Context, hotelID string) error {
	// Get hotel
	hotel, err := s.hotelService.GetHotel(ctx, hotelID)
	if err != nil {
		return fmt.Errorf("failed to get hotel: %w", err)
	}

	// Check admin flag - if disabled, don't process
	if hotel.AdminFlag {
		log.Printf("Hotel %s is disabled by admin flag, skipping", hotelID)
		return nil
	}

	// Crawl reviews from all sources
	log.Printf("Crawling reviews for hotel %s (%s)", hotelID, hotel.Name)
	reviewCount, err := s.reviewCrawler.CrawlAllSources(ctx, hotel)
	if err != nil {
		log.Printf("Error crawling reviews: %v", err)
		// Continue anyway with existing reviews
	} else {
		log.Printf("Crawled %d new reviews", reviewCount)
	}

	// Get all review texts
	reviewTexts, err := s.reviewCrawler.GetReviewTexts(ctx, hotelID)
	if err != nil {
		return fmt.Errorf("failed to get review texts: %w", err)
	}

	if len(reviewTexts) == 0 {
		log.Printf("No reviews found for hotel %s, skipping LLM analysis", hotelID)
		return nil
	}

	log.Printf("Analyzing %d reviews with LLM for hotel %s", len(reviewTexts), hotelID)

	// Analyze with LLM
	qualityAnalysis, quietAnalysis, err := s.llmService.AnalyzeHotelReviews(ctx, reviewTexts)
	if err != nil {
		return fmt.Errorf("failed to analyze reviews: %w", err)
	}

	// Determine final recommendation
	recommended := qualityAnalysis.Recommended && quietAnalysis.IsQuiet && qualityAnalysis.Score >= 0.7 && quietAnalysis.Score >= 0.7

	// Create recommendation record
	recommendation := &models.HotelRecommendation{
		HotelID:            hotelID,
		QualityScore:       qualityAnalysis.Score,
		QualityConfidence:  qualityAnalysis.Confidence,
		QualityReasoning:   qualityAnalysis.Reasoning,
		QuietScore:         quietAnalysis.Score,
		QuietConfidence:    quietAnalysis.Confidence,
		QuietReasoning:     quietAnalysis.Reasoning,
		OverallRecommended: recommended,
		ReviewsAnalyzed:    len(reviewTexts),
		LLMModel:           s.llmService.provider.GetModelName(),
	}

	// Create summary
	var summaryParts []string
	summaryParts = append(summaryParts, fmt.Sprintf("Quality: %.1f/1.0 (confidence: %.1f%%)", qualityAnalysis.Score, qualityAnalysis.Confidence*100))
	summaryParts = append(summaryParts, fmt.Sprintf("Quiet: %.1f/1.0 (confidence: %.1f%%)", quietAnalysis.Score, quietAnalysis.Confidence*100))
	if recommended {
		summaryParts = append(summaryParts, "Overall: RECOMMENDED")
	} else {
		summaryParts = append(summaryParts, "Overall: NOT RECOMMENDED")
	}
	recommendation.RecommendationSummary = strings.Join(summaryParts, " | ")

	// Save recommendation
	if err := s.hotelService.SaveRecommendation(ctx, recommendation); err != nil {
		return fmt.Errorf("failed to save recommendation: %w", err)
	}

	// Update hotel fields
	importantNote := fmt.Sprintf("Quality: %s | Quiet: %s", qualityAnalysis.Reasoning, quietAnalysis.Reasoning)
	if err := s.hotelService.UpdateRecommendationFields(ctx, hotelID, recommended, qualityAnalysis.Score >= 0.7, quietAnalysis.Score >= 0.7, importantNote); err != nil {
		return fmt.Errorf("failed to update hotel fields: %w", err)
	}

	log.Printf("Successfully processed recommendations for hotel %s", hotelID)
	return nil
}

// ProcessAllHotels processes recommendations for all hotels
func (s *RecommendationService) ProcessAllHotels(ctx context.Context) error {
	// Get all hotel IDs
	hotelIDs, err := s.hotelService.GetHotelIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get hotel IDs: %w", err)
	}

	log.Printf("Processing recommendations for %d hotels", len(hotelIDs))

	successCount := 0
	errorCount := 0

	for _, hotelID := range hotelIDs {
		if err := s.ProcessHotelRecommendations(ctx, hotelID); err != nil {
			log.Printf("Error processing hotel %s: %v", hotelID, err)
			errorCount++
			continue
		}
		successCount++
	}

	log.Printf("Completed: %d successful, %d errors", successCount, errorCount)
	return nil
}
