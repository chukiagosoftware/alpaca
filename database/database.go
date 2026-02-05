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
	CREATE TABLE IF NOT EXISTS hotels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hotel_id TEXT UNIQUE NOT NULL,
		type TEXT,
		chain_code TEXT,
		dupe_id INTEGER,
		name TEXT,
		iata_code TEXT,
		address TEXT,
		geo_code TEXT,
		distance TEXT,
		last_update TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

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

	CREATE INDEX IF NOT EXISTS idx_hotels_hotel_id ON hotels(hotel_id);
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
