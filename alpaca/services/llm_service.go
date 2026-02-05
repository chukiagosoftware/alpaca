package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// LLMProvider defines the interface for LLM services
type LLMProvider interface {
	AnalyzeQuality(ctx context.Context, reviews []string) (*QualityAnalysis, error)
	AnalyzeQuiet(ctx context.Context, reviews []string) (*QuietAnalysis, error)
	GetModelName() string
}

// QualityAnalysis represents the LLM's analysis of hotel quality
type QualityAnalysis struct {
	Score       float64 `json:"score"`      // 0.0 to 1.0
	Confidence  float64 `json:"confidence"` // 0.0 to 1.0
	Reasoning   string  `json:"reasoning"`
	Recommended bool    `json:"recommended"`
}

// QuietAnalysis represents the LLM's analysis of hotel quietness
type QuietAnalysis struct {
	Score      float64 `json:"score"`      // 0.0 to 1.0
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
	Reasoning  string  `json:"reasoning"`
	IsQuiet    bool    `json:"isQuiet"`
}

// LLMService handles LLM interactions
type LLMService struct {
	provider LLMProvider
}

// NewLLMService creates a new LLM service
func NewLLMService(provider LLMProvider) *LLMService {
	return &LLMService{provider: provider}
}

// AnalyzeHotelReviews analyzes reviews for quality and quietness
func (s *LLMService) AnalyzeHotelReviews(ctx context.Context, reviews []string) (*QualityAnalysis, *QuietAnalysis, error) {
	// Batch reviews for efficiency (process in chunks)
	const maxReviewsPerBatch = 50
	var allReviewTexts []string

	for i, review := range reviews {
		if i >= maxReviewsPerBatch {
			break
		}
		allReviewTexts = append(allReviewTexts, review)
	}

	// Analyze quality and quiet in parallel
	qualityChan := make(chan *QualityAnalysis, 1)
	quietChan := make(chan *QuietAnalysis, 1)
	errChan := make(chan error, 2)

	go func() {
		analysis, err := s.provider.AnalyzeQuality(ctx, allReviewTexts)
		if err != nil {
			errChan <- fmt.Errorf("quality analysis error: %w", err)
			return
		}
		qualityChan <- analysis
	}()

	go func() {
		analysis, err := s.provider.AnalyzeQuiet(ctx, allReviewTexts)
		if err != nil {
			errChan <- fmt.Errorf("quiet analysis error: %w", err)
			return
		}
		quietChan <- analysis
	}()

	// Wait for both analyses
	var quality *QualityAnalysis
	var quiet *QuietAnalysis
	errors := 0

	for errors < 2 {
		select {
		case q := <-qualityChan:
			quality = q
			errors++
		case q := <-quietChan:
			quiet = q
			errors++
		case err := <-errChan:
			return nil, nil, err
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	return quality, quiet, nil
}

// GetQualityPrompt returns the prompt for quality analysis
func GetQualityPrompt(reviews []string) string {
	reviewsText := strings.Join(reviews, "\n\n---\n\n")

	return fmt.Sprintf(`You are analyzing hotel reviews to determine if a hotel has "Quality" - meaning it provides excellent service, cleanliness, amenities, and overall guest satisfaction.

Review the following hotel reviews and determine:
1. Quality Score (0.0 to 1.0): How high is the quality based on reviews?
2. Confidence (0.0 to 1.0): How confident are you in this assessment?
3. Reasoning: Brief explanation of your assessment
4. Recommended: Should this hotel be recommended based on quality? (true/false)

Quality indicators (POSITIVE):
- Excellent service, staff helpfulness
- Cleanliness, well-maintained
- Good amenities, facilities
- Comfortable rooms, good beds
- Good value for money
- Positive guest experiences
- Professional management
- Good location (if mentioned positively)

Quality disqualifiers (NEGATIVE):
- Poor service, unhelpful staff
- Dirty, unclean conditions
- Broken amenities, poor maintenance
- Uncomfortable rooms, bad beds
- Poor value, overpriced
- Negative guest experiences
- Unprofessional management
- Safety concerns

Reviews to analyze:
%s

Respond in JSON format:
{
  "score": 0.85,
  "confidence": 0.9,
  "reasoning": "Most reviews mention excellent service and cleanliness. Some minor complaints about slow WiFi.",
  "recommended": true
}`, reviewsText)
}

// GetQuietPrompt returns the prompt for quietness analysis
func GetQuietPrompt(reviews []string) string {
	reviewsText := strings.Join(reviews, "\n\n---\n\n")

	return fmt.Sprintf(`You are analyzing hotel reviews to determine if a hotel is "Quiet" - meaning it provides a peaceful, noise-free environment for guests.

Review the following hotel reviews and determine:
1. Quiet Score (0.0 to 1.0): How quiet is the hotel based on reviews?
2. Confidence (0.0 to 1.0): How confident are you in this assessment?
3. Reasoning: Brief explanation of your assessment
4. IsQuiet: Is this hotel quiet? (true/false)

Quiet indicators (POSITIVE - mentions of quiet/peaceful):
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

Quiet disqualifiers (NEGATIVE - mentions of noise):
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

Reviews to analyze:
%s

Respond in JSON format:
{
  "score": 0.75,
  "confidence": 0.85,
  "reasoning": "Most reviews mention peaceful location away from traffic. One review mentioned thin walls between rooms.",
  "isQuiet": true
}`, reviewsText)
}

// OpenAIProvider implements LLMProvider using OpenAI API
type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4"
	}
	return &OpenAIProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// GetModelName returns the model name
func (p *OpenAIProvider) GetModelName() string {
	return p.model
}

// AnalyzeQuality analyzes reviews for quality
func (p *OpenAIProvider) AnalyzeQuality(ctx context.Context, reviews []string) (*QualityAnalysis, error) {
	prompt := GetQualityPrompt(reviews)
	result, err := p.callLLM(ctx, prompt, "quality")
	if err != nil {
		return nil, err
	}
	return result.(*QualityAnalysis), nil
}

// AnalyzeQuiet analyzes reviews for quietness
func (p *OpenAIProvider) AnalyzeQuiet(ctx context.Context, reviews []string) (*QuietAnalysis, error) {
	prompt := GetQuietPrompt(reviews)
	result, err := p.callLLM(ctx, prompt, "quiet")
	if err != nil {
		return nil, err
	}
	return result.(*QuietAnalysis), nil
}

// callLLM makes a call to OpenAI API
func (p *OpenAIProvider) callLLM(ctx context.Context, prompt, analysisType string) (interface{}, error) {
	url := "https://api.openai.com/v1/chat/completions"

	payload := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.3,
		"max_tokens":  500,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content

	// Parse JSON response
	if analysisType == "quality" {
		var analysis QualityAnalysis
		if err := json.Unmarshal([]byte(content), &analysis); err != nil {
			return nil, fmt.Errorf("failed to parse quality analysis: %w", err)
		}
		return &analysis, nil
	} else {
		var analysis QuietAnalysis
		if err := json.Unmarshal([]byte(content), &analysis); err != nil {
			return nil, fmt.Errorf("failed to parse quiet analysis: %w", err)
		}
		return &analysis, nil
	}
}

// ClaudeProvider implements LLMProvider using Anthropic Claude API
type ClaudeProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(apiKey string) *ClaudeProvider {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-3-opus-20240229"
	}
	return &ClaudeProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// GetModelName returns the model name
func (p *ClaudeProvider) GetModelName() string {
	return p.model
}

// AnalyzeQuality analyzes reviews for quality
func (p *ClaudeProvider) AnalyzeQuality(ctx context.Context, reviews []string) (*QualityAnalysis, error) {
	// Similar implementation to OpenAI but using Claude API
	// This is a placeholder - implement Claude API calls
	log.Printf("Claude quality analysis not yet implemented")
	return &QualityAnalysis{Score: 0.5, Confidence: 0.5, Reasoning: "Not implemented", Recommended: false}, nil
}

// AnalyzeQuiet analyzes reviews for quietness
func (p *ClaudeProvider) AnalyzeQuiet(ctx context.Context, reviews []string) (*QuietAnalysis, error) {
	// Similar implementation to OpenAI but using Claude API
	log.Printf("Claude quiet analysis not yet implemented")
	return &QuietAnalysis{Score: 0.5, Confidence: 0.5, Reasoning: "Not implemented", IsQuiet: false}, nil
}

// GrokProvider implements LLMProvider using X (Twitter) Grok API
type GrokProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewGrokProvider creates a new Grok provider
func NewGrokProvider(apiKey string) *GrokProvider {
	if apiKey == "" {
		apiKey = os.Getenv("GROK_API_KEY")
	}
	model := os.Getenv("GROK_MODEL")
	if model == "" {
		model = "grok-beta"
	}
	return &GrokProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// GetModelName returns the model name
func (p *GrokProvider) GetModelName() string {
	return p.model
}

// AnalyzeQuality analyzes reviews for quality
func (p *GrokProvider) AnalyzeQuality(ctx context.Context, reviews []string) (*QualityAnalysis, error) {
	// Similar implementation to OpenAI but using Grok API
	log.Printf("Grok quality analysis not yet implemented")
	return &QualityAnalysis{Score: 0.5, Confidence: 0.5, Reasoning: "Not implemented", Recommended: false}, nil
}

// AnalyzeQuiet analyzes reviews for quietness
func (p *GrokProvider) AnalyzeQuiet(ctx context.Context, reviews []string) (*QuietAnalysis, error) {
	// Similar implementation to OpenAI but using Grok API
	log.Printf("Grok quiet analysis not yet implemented")
	return &QuietAnalysis{Score: 0.5, Confidence: 0.5, Reasoning: "Not implemented", IsQuiet: false}, nil
}
