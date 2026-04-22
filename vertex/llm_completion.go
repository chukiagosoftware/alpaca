package vertex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/genai"
)

type GeminiProvider struct {
	client genai.Client
	model  string
	config Config
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) CheckQuerySafety(ctx context.Context, question string) (bool, error) {
	fullPrompt := fmt.Sprintf("%s\n\nUser query: %s\n\nAnswer with only the word YES or NO.", p.config.SecurityPrompt, question)
	resp, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(fullPrompt), &genai.GenerateContentConfig{})
	if err != nil {
		return false, err
	}
	text := strings.ToLower(strings.TrimSpace(resp.Text()))
	return strings.Contains(text, "yes"), nil
}

func (p *GeminiProvider) PromptCompletion(ctx context.Context, question string, results []map[string]any) (CompletionResult, error) {
	promptText := buildCompletionPrompt(p.config.Prompt, question, results)
	resp, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(promptText), &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: completionJSONSchema(),
		Temperature:        float32Ptr(0.2),
		MaxOutputTokens:    16384,
	})
	if err != nil {
		return CompletionResult{}, err
	}

	text := resp.Text()
	usage := TokenUsage{}
	if resp.UsageMetadata != nil {
		usage.PromptTokens = int(resp.UsageMetadata.PromptTokenCount)
		if resp.UsageMetadata.CandidatesTokenCount > 0 {
			usage.CompletionTokens = int(resp.UsageMetadata.CandidatesTokenCount)
		}
		usage.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
	}

	return CompletionResult{
		Content: text,
		Usage:   usage,
		Model:   p.model,
	}, nil
}

type GrokProvider struct {
	apiKey  string
	baseURL string
	model   string
	config  Config
	client  *http.Client
}

func (p *GrokProvider) Name() string { return "grok" }

func (p *GrokProvider) CheckQuerySafety(ctx context.Context, question string) (bool, error) {
	if p.apiKey == "" {
		return false, fmt.Errorf("grok api key missing")
	}
	url := strings.TrimRight(p.baseURL, "/") + "/v1/chat/completions"
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Answer only YES or NO. This is a hotel review relevance safety check."},
			{"role": "user", "content": fmt.Sprintf("%s\n\nUser query: %s\n\nAnswer with only the word YES or NO.", p.config.SecurityPrompt, question)},
		},
		"temperature": 0.0,
		"max_tokens":  32,
	}
	content, err := p.doChatCompletion(ctx, url, body)
	if err != nil {
		return false, err
	}
	text := strings.ToLower(strings.TrimSpace(content))
	return strings.Contains(text, "yes"), nil
}

func (p *GrokProvider) PromptCompletion(ctx context.Context, question string, results []map[string]any) (CompletionResult, error) {
	if p.apiKey == "" {
		return CompletionResult{}, fmt.Errorf("grok api key missing")
	}
	url := strings.TrimRight(p.baseURL, "/") + "/v1/chat/completions"
	promptText := buildCompletionPrompt(p.config.Prompt, question, results)
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Return only valid JSON matching the schema."},
			{"role": "user", "content": promptText},
		},
		"temperature": 0.2,
		"max_tokens":  16384,
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "hotel_review_completion",
				"schema": completionJSONSchema(),
				"strict": true,
			},
		},
	}
	content, err := p.doChatCompletion(ctx, url, body)
	if err != nil {
		return CompletionResult{}, err
	}
	return CompletionResult{
		Content: content,
		Usage:   TokenUsage{},
		Model:   p.model,
	}, nil
}

func (p *GrokProvider) doChatCompletion(ctx context.Context, url string, body map[string]any) (string, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("grok completion failed: %s: %s", resp.Status, string(respBytes))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("grok returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	config  Config
	client  *http.Client
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) CheckQuerySafety(ctx context.Context, question string) (bool, error) {
	if p.apiKey == "" {
		return false, fmt.Errorf("openai api key missing")
	}
	url := strings.TrimRight(p.baseURL, "/") + "/v1/chat/completions"
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Answer only YES or NO. This is a hotel review relevance safety check."},
			{"role": "user", "content": fmt.Sprintf("%s\n\nUser query: %s\n\nAnswer with only the word YES or NO.", p.config.SecurityPrompt, question)},
		},
		"temperature": 0.0,
		"max_tokens":  32,
	}
	content, err := p.doChatCompletion(ctx, url, body)
	if err != nil {
		return false, err
	}
	text := strings.ToLower(strings.TrimSpace(content))
	return strings.Contains(text, "yes"), nil
}

func (p *OpenAIProvider) PromptCompletion(ctx context.Context, question string, results []map[string]any) (CompletionResult, error) {
	if p.apiKey == "" {
		return CompletionResult{}, fmt.Errorf("openai api key missing")
	}
	url := strings.TrimRight(p.baseURL, "/") + "/v1/chat/completions"
	promptText := buildCompletionPrompt(p.config.Prompt, question, results)
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Return only valid JSON matching the schema."},
			{"role": "user", "content": promptText},
		},
		"temperature": 0.2,
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "hotel_review_completion",
				"schema": completionJSONSchema(),
				"strict": true,
			},
		},
	}
	content, err := p.doChatCompletion(ctx, url, body)
	if err != nil {
		return CompletionResult{}, err
	}
	return CompletionResult{
		Content: content,
		Usage:   TokenUsage{},
		Model:   p.model,
	}, nil
}

func (p *OpenAIProvider) doChatCompletion(ctx context.Context, url string, body map[string]any) (string, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai completion failed: %s: %s", resp.Status, string(respBytes))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
