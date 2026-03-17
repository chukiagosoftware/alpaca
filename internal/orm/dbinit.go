package orm

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/chukiagosoftware/alpaca/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DB wraps the GORM database connection
type DB struct {
	*gorm.DB
}

// NewDatabase creates a new GORM SQLite database connection and runs migrations
func NewDatabase() (*DB, error) {
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			return nil, fmt.Errorf("failed to get current file path")
		}
		projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
		dbPath = filepath.Join(projectRoot, "alpaca.db")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	log.Printf("Connected to SQLite database: %s", dbPath)

	// Run auto-migrations for kept tables
	db = db.Debug() //Gorm detailed logs
	if err := db.AutoMigrate(
		&models.Hotel{},
		&models.HotelReview{},
		&models.AirportCity{},
		&models.AmadeusTestDetailedDataUnavailable{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migrated successfully with GORM")
	return &DB{db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
