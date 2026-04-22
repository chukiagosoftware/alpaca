package vertex

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"google.golang.org/genai"
)

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type CompletionResult struct {
	Content string     `json:"content"`
	Usage   TokenUsage `json:"usage"`
	Model   string     `json:"model,omitempty"`
}

type LLMChoice string

const (
	LLMChoiceAuto   LLMChoice = "auto"
	LLMChoiceGemini LLMChoice = "gemini"
	LLMChoiceGrok   LLMChoice = "grok"
	LLMChoiceOpenAI LLMChoice = "openai"
)

type LLMProvider interface {
	Name() string
	CheckQuerySafety(ctx context.Context, question string) (bool, error)
	PromptCompletion(ctx context.Context, question string, results []map[string]any) (CompletionResult, error)
}

type CompletionRouter struct {
	config    *Config
	providers map[LLMChoice]LLMProvider
}

func NewCompletionRouter(config *Config, geminiClient genai.Client) (*CompletionRouter, error) {
	grokKey := firstNonEmpty(config.GrokAPIKey, os.Getenv("GROK_API_KEY"))
	openAIKey := firstNonEmpty(config.OpenAIAPIKey, os.Getenv("OPENAI_API_KEY"))

	providers := map[LLMChoice]LLMProvider{
		LLMChoiceGemini: &GeminiProvider{
			client: geminiClient,
			model:  firstNonEmpty(config.GeminiModel, "gemini-2.5-flash-lite"),
			config: *config,
		},
	}

	if grokKey != "" {
		providers[LLMChoiceGrok] = &GrokProvider{
			apiKey:  grokKey,
			baseURL: "https://api.x.ai",
			model:   firstNonEmpty(config.GrokModel, "grok-4-1-fast-non-reasoning"),
			config:  *config,
			client:  &http.Client{},
		}
	} else {
		log.Println("Grok API key not configured, Grok provider disabled")
	}

	if openAIKey != "" {
		providers[LLMChoiceOpenAI] = &OpenAIProvider{
			apiKey:  openAIKey,
			baseURL: "https://api.openai.com",
			model:   firstNonEmpty(config.OpenAIModel, "gpt-4.1-mini"),
			config:  *config,
			client:  &http.Client{},
		}
	} else {
		log.Println("OpenAI API key not configured, OpenAI provider disabled")
	}

	return &CompletionRouter{
		config:    config,
		providers: providers,
	}, nil
}

func (r *CompletionRouter) CheckQuerySafety(ctx context.Context, input SearchInput) (bool, error) {
	chain := r.resolveChain(input.PreferredModel)
	var errs []string
	for _, provider := range chain {
		ok, err := provider.CheckQuerySafety(ctx, input.Question)
		if err == nil {
			return ok, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", provider.Name(), err))
		if !isRetryableLLMError(err) {
			break
		}
	}
	return false, fmt.Errorf("all safety providers failed: %s", strings.Join(errs, " | "))
}

func (r *CompletionRouter) PromptCompletion(ctx context.Context, input SearchInput, results []map[string]any) (CompletionResult, error) {
	chain := r.resolveChain(input.PreferredModel)
	var errs []string
	for _, provider := range chain {
		resp, err := provider.PromptCompletion(ctx, input.Question, results)
		if err == nil {
			return resp, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", provider.Name(), err))
		if !isRetryableLLMError(err) {
			break
		}
	}
	return CompletionResult{}, fmt.Errorf("all completion providers failed: %s", strings.Join(errs, " | "))
}

func (r *CompletionRouter) resolveChain(model string) []LLMProvider {
	norm := strings.ToLower(strings.TrimSpace(model))
	if norm == "" || norm == string(LLMChoiceAuto) {
		norm = strings.ToLower(strings.TrimSpace(r.config.PreferredModel))
		if norm == "" {
			norm = string(LLMChoiceGemini)
		}
	}

	var ordered []LLMProvider
	switch LLMChoice(norm) {
	case LLMChoiceGrok:
		ordered = []LLMProvider{r.providers[LLMChoiceGrok], r.providers[LLMChoiceGemini], r.providers[LLMChoiceOpenAI]}
	case LLMChoiceOpenAI:
		ordered = []LLMProvider{r.providers[LLMChoiceOpenAI], r.providers[LLMChoiceGemini], r.providers[LLMChoiceGrok]}
	default:
		ordered = []LLMProvider{r.providers[LLMChoiceGemini], r.providers[LLMChoiceGrok], r.providers[LLMChoiceOpenAI]}
	}

	var chain []LLMProvider
	for _, p := range ordered {
		if p != nil {
			chain = append(chain, p)
		}
	}
	return chain
}

func isRetryableLLMError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "timeout") || strings.Contains(msg, "temporarily") ||
		strings.Contains(msg, "unavailable") || strings.Contains(msg, "resource exhausted")
}

func completionJSONSchema() map[string]any {
	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"Hotel":           map[string]any{"type": "string"},
				"City":            map[string]any{"type": "string"},
				"Review":          map[string]any{"type": "string"},
				"Rating":          map[string]any{"type": "number"},
				"Distance":        map[string]any{"type": "number"},
				"Address":         map[string]any{"type": "string"},
				"google_maps_uri": map[string]any{"type": "string"},
				"photo_name":      map[string]any{"type": "string"},
			},
			"required": []string{"Hotel", "City", "Review", "Rating", "Distance", "Address"},
		},
	}
}

func buildCompletionPrompt(systemPrompt, question string, results []map[string]any) string {
	var reviewContext strings.Builder
	for i, r := range results {
		reviewContext.WriteString(fmt.Sprintf(
			"Review %d:\nHotel: %v\nCity: %v\nReview: %v\nRating: %v\nDistance: %.3f\nAddress: %v\nGoogleMapsURI: %v\nPhotoName: %v\n\n",
			i+1, r["hotel_name"], r["city"], r["review_text"], r["rating"], r["distance"],
			r["street_address"], r["google_maps_uri"], r["photo_name"],
		))
	}
	return fmt.Sprintf(`%s

User Question: %s

Important: If a review has a google_maps_uri or photo_name, you MUST include them in the JSON output.

Reviews:
%s`, systemPrompt, question, reviewContext.String())
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
