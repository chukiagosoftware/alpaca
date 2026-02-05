package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the sql.DB connection
type DB struct {
	*sql.DB
}

// NewDatabase creates a new SQLite database connection
func NewDatabase() (*DB, error) {
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		// Get the directory where this database.go file is located
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			return nil, fmt.Errorf("failed to get current file path")
		}

		// database.go is in the database/ directory, so go up one level to project root
		dbDir := filepath.Dir(filename)    // database/
		projectRoot := filepath.Dir(dbDir) // project root
		dbPath = filepath.Join(projectRoot, "alpaca.db")
	}

	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to SQLite database: %s", dbPath)

	// Create tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &DB{db}, nil
}

// createTables creates all necessary tables
func createTables(db *sql.DB) error {
	schema := `
	-- Consolidated hotels table with all sources
	CREATE TABLE IF NOT EXISTS hotels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hotel_id TEXT UNIQUE NOT NULL,
		source TEXT NOT NULL,  -- 'amadeus', 'expedia', 'tripadvisor', 'google', etc.
		source_hotel_id TEXT,  -- Original ID from source
		name TEXT NOT NULL,
		city TEXT,
		country TEXT,
		latitude REAL,
		longitude REAL,
		street_address TEXT,
		postal_code TEXT,
		phone TEXT,
		website TEXT,
		email TEXT,
		
		-- Ratings from different sources
		amadeus_rating REAL,
		expedia_rating REAL,
		tripadvisor_rating REAL,
		google_rating REAL,
		booking_rating REAL,
		
		-- Recommendation fields
		recommended INTEGER DEFAULT 0,  -- Boolean: 0 = false, 1 = true
		admin_flag INTEGER DEFAULT 0,   -- Boolean: Admin override (0 = enabled, 1 = disabled)
		quality INTEGER DEFAULT 0,      -- Boolean: Has quality
		quiet INTEGER DEFAULT 0,        -- Boolean: Is quiet
		important_note TEXT,            -- Notes about recommendation calculation
		
		-- Original Amadeus fields (for backward compatibility)
		type TEXT,
		chain_code TEXT,
		dupe_id INTEGER,
		iata_code TEXT,
		address_json TEXT,   -- JSON stored as TEXT
		geo_code_json TEXT,  -- JSON stored as TEXT
		distance_json TEXT,  -- JSON stored as TEXT
		
		last_update TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Hotel reviews from multiple sources
	CREATE TABLE IF NOT EXISTS hotel_reviews (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hotel_id TEXT NOT NULL,
		source TEXT NOT NULL,  -- 'tripadvisor', 'google', 'expedia', 'booking', 'hotel_website', etc.
		source_review_id TEXT, -- Original review ID from source
		reviewer_name TEXT,
		reviewer_location TEXT,
		rating REAL,
		review_text TEXT NOT NULL,
		review_date DATETIME,
		verified INTEGER DEFAULT 0,  -- Boolean: Is verified review
		helpful_count INTEGER DEFAULT 0,
		room_type TEXT,
		travel_type TEXT,  -- 'business', 'leisure', 'family', etc.
		stay_date DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (hotel_id) REFERENCES hotels(hotel_id),
		UNIQUE(hotel_id, source, source_review_id)
	);

	-- LLM-processed recommendations
	CREATE TABLE IF NOT EXISTS hotel_recommendations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hotel_id TEXT UNIQUE NOT NULL,
		quality_score REAL,      -- 0.0 to 1.0
		quality_confidence REAL, -- 0.0 to 1.0
		quality_reasoning TEXT,
		quiet_score REAL,        -- 0.0 to 1.0
		quiet_confidence REAL,   -- 0.0 to 1.0
		quiet_reasoning TEXT,
		overall_recommended INTEGER DEFAULT 0,  -- Boolean
		recommendation_summary TEXT,
		reviews_analyzed INTEGER DEFAULT 0,
		llm_model TEXT,  -- 'gpt-4', 'claude', 'grok', etc.
		processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (hotel_id) REFERENCES hotels(hotel_id)
	);

	-- Original Amadeus tables (kept for backward compatibility)
	CREATE TABLE IF NOT EXISTS hotel_search_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hotel_id TEXT UNIQUE NOT NULL,
		type TEXT,
		chain_code TEXT,
		dupe_id INTEGER,
		name TEXT,
		rating INTEGER,
		official_rating INTEGER,
		description TEXT,
		media TEXT,
		amenities TEXT,
		address TEXT,
		contact TEXT,
		policies TEXT,
		available INTEGER DEFAULT 0,
		offers TEXT,
		self TEXT,
		hotel_distance TEXT,
		last_update TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (hotel_id) REFERENCES hotels(hotel_id)
	);

	CREATE TABLE IF NOT EXISTS hotel_ratings_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hotel_id TEXT UNIQUE NOT NULL,
		type TEXT,
		number_of_reviews INTEGER,
		number_of_ratings INTEGER,
		overall_rating INTEGER,
		sentiments TEXT,
		last_update TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (hotel_id) REFERENCES hotels(hotel_id)
	);

	CREATE TABLE IF NOT EXISTS invalid_hotel_search_ids (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hotel_id TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_hotels_hotel_id ON hotels(hotel_id);
	CREATE INDEX IF NOT EXISTS idx_hotels_source ON hotels(source);
	CREATE INDEX IF NOT EXISTS idx_hotels_city_country ON hotels(city, country);
	CREATE INDEX IF NOT EXISTS idx_hotels_recommended ON hotels(recommended);
	CREATE INDEX IF NOT EXISTS idx_hotels_admin_flag ON hotels(admin_flag);
	CREATE INDEX IF NOT EXISTS idx_hotels_quality ON hotels(quality);
	CREATE INDEX IF NOT EXISTS idx_hotels_quiet ON hotels(quiet);
	
	CREATE INDEX IF NOT EXISTS idx_hotel_reviews_hotel_id ON hotel_reviews(hotel_id);
	CREATE INDEX IF NOT EXISTS idx_hotel_reviews_source ON hotel_reviews(source);
	CREATE INDEX IF NOT EXISTS idx_hotel_reviews_rating ON hotel_reviews(rating);
	
	CREATE INDEX IF NOT EXISTS idx_hotel_recommendations_hotel_id ON hotel_recommendations(hotel_id);
	CREATE INDEX IF NOT EXISTS idx_hotel_search_data_hotel_id ON hotel_search_data(hotel_id);
	CREATE INDEX IF NOT EXISTS idx_hotel_ratings_data_hotel_id ON hotel_ratings_data(hotel_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
