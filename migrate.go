package main

import (
	"log"

	"github.com/chukiagosoftware/alpaca/internal/orm"
)

func main() {
	// Initialize the database (this will create tables and run migrations like adding state_code)
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Database initialized successfully .")
}
