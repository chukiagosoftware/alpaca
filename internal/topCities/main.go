package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gorm.io/gorm"

	"github.com/chukiagosoftware/alpaca/internal/orm"
	"github.com/chukiagosoftware/alpaca/models"
)

func readTopCities(db *gorm.DB) error {

	_, currentFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(currentFile)
	var cities []models.City

	err := filepath.Walk(moduleRoot, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".txt") {
			return nil
		}

		file, err := os.Open(filepath.Join(moduleRoot, info.Name()))
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(file)

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines) // Split by lines is default
		for scanner.Scan() {
			row := strings.Split(scanner.Text(), ",")
			if len(row) != 2 {
				log.Printf("Invalid row: %s", scanner.Text())
				continue
			}
			cities = append(cities, models.City{Name: row[0], Country: row[1]})
			// log.Printf("Found city: %s, %s", row[0], row[1])
		}
		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Found %d cities\n", len(cities))

	err = db.Create(&cities).Error
	if err != nil {
		return err
	}

	return nil
}

func main() {
	db, err := orm.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	err = readTopCities(db.DB)

	if err != nil {
		log.Fatalf("Failed to read top cities: %v", err)
	}
}
