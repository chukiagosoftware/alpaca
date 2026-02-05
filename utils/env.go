package utils

import (
	"log"

	"github.com/joho/godotenv"
)

// LoadEnv tries to load .env file from multiple common locations
// This allows the same code to work whether running from project root or subdirectories
func LoadEnv() {
	envPaths := []string{
		".env",       // Current directory
		"../.env",    // Parent directory
		"../../.env", // Grandparent directory
	}

	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env file from: %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Printf("Warning: No .env file found in any of the expected locations")
		log.Printf("Make sure environment variables are set manually")
	}
}

// LoadEnvOverride loads .env file but allows environment variables to override .env values
// This is useful for production where you want to override .env with actual environment variables
func LoadEnvOverride() {
	envPaths := []string{
		".env",       // Current directory
		"../.env",    // Parent directory
		"../../.env", // Grandparent directory
	}

	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Overload(path); err == nil {
			log.Printf("Loaded .env file from: %s (with override)", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Printf("Warning: No .env file found in any of the expected locations")
		log.Printf("Make sure environment variables are set manually")
	}
}
