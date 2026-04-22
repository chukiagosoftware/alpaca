package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
)

func parseCompletionJSON(completionStr string) ([]map[string]any, error) {
	trimmed := strings.TrimSpace(completionStr)
	if trimmed == "" || trimmed == "null" {
		return []map[string]any{}, nil
	}

	dirtyRegexp := regexp.MustCompile("(?s)^```(?:json)?\\s*|\\s*```$")
	clean := dirtyRegexp.ReplaceAllString(trimmed, "")

	var err error
	var reviews []map[string]any
	if err = json.Unmarshal([]byte(clean), &reviews); err == nil {
		log.Println("Successfully parsed LLM returned JSON")
		return reviews, nil
	}

	log.Printf("Failed to unmarshal LLM returned JSON. Trying backtick removal %v\n", err)
	noBackTick := strings.ReplaceAll(trimmed, "```", "")
	noFrontTick := strings.ReplaceAll(noBackTick, "json```", "")
	if err = json.Unmarshal([]byte(noFrontTick), &reviews); err == nil {
		log.Println("Successfully parsed LLM returned JSON after removing backticks")
		return reviews, nil
	}
	log.Printf("Failed to parse LLM returned JSON after backtick removal: %v\n", err)
	return nil, err
}

func enrichReviewWithGoogleMedia(review *map[string]any, apiKey string) {
	if review == nil {
		return
	}

	// 1. Map - google_maps_uri is already a complete usable URL
	if uriIface, ok := (*review)["google_maps_uri"]; ok {
		if uri, ok := uriIface.(string); ok && uri != "" {
			(*review)["map_url"] = uri
		}
	}

	// 2. Photo - New Places API v2 (exact format you confirmed)
	if photoNameIface, ok := (*review)["photo_name"]; ok {
		if photoName, ok := photoNameIface.(string); ok && photoName != "" {
			thumbURL := fmt.Sprintf("https://places.googleapis.com/v1/%s/media?maxWidthPx=120&key=%s", photoName, apiKey)
			fullURL := fmt.Sprintf("https://places.googleapis.com/v1/%s/media?maxWidthPx=800&key=%s", photoName, apiKey)

			(*review)["photo_thumb"] = thumbURL
			(*review)["photo_full"] = fullURL
		}
	}
}
