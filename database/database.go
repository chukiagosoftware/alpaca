package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewDatabase creates a GORM database connection based on environment variables
func NewDatabase() (*gorm.DB, error) {
	dbType := os.Getenv("DATABASE_TYPE")

	switch dbType {
	case "postgres":
		return NewPostgresDB()
	case "sqlite":
		return NewSQLiteDB()
	default:
		// Default to SQLite if not specified
		return NewSQLiteDB()
	}
}

func NewSQLiteDB() (*gorm.DB, error) {
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

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to SQLite database: %s", dbPath)
	return db, nil
}

func NewPostgresDB() (*gorm.DB, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required for PostgreSQL")
	}

	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to PostgreSQL database")
	return db, nil
}
