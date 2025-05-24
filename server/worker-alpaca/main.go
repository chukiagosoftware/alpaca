package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/edamsoft-sre/alpaca/models"
    "github.com/edamsoft-sre/alpaca/repository"
    "github.com/joho/godotenv"
)

type HotelsAPIResponse struct {
    Hotels   []models.Hotel `json:"hotels"`
    NextPage string         `json:"next_page"` // "" or "END" or null means end
}

func fetchHotelsPaginated(ctx context.Context, baseURL, token string) error {
    nextPage := ""
    for {
        url := baseURL
        if nextPage != "" {
            url = fmt.Sprintf("%s?next_page=%s", baseURL, nextPage)
        }
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        req.Header.Set("Authorization", "Bearer "+token)
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return fmt.Errorf("error fetching %s: %w", url, err)
        }
        defer resp.Body.Close()

        var apiResp HotelsAPIResponse
        if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
            return fmt.Errorf("decode error: %w", err)
        }

        for _, hotel := range apiResp.Hotels {
            if err := repository.InsertHotel(ctx, &hotel); err != nil {
                log.Printf("DB error: %v", err)
            }
        }

        // Detect end of stream
        if apiResp.NextPage == "" || apiResp.NextPage == "END" {
            break
        }
        nextPage = apiResp.NextPage
    }
    return nil
}

func main() {
    _ = godotenv.Load()
    apiToken := os.Getenv("API_TOKEN")
    baseURL := os.Getenv("HOTEL_API_URL") // e.g. "https://api.example.com/hotels"

    ctx := context.Background()
    if err := fetchHotelsPaginated(ctx, baseURL, apiToken); err != nil {
        log.Fatal(err)
    }
}