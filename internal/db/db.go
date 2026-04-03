package db

import (
	"fmt"
	"gopickup/internal/config"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg *config.Config) {
	var dialector gorm.Dialector

	if cfg.DBDriver == "sqlite" {
		dialector = sqlite.Open(cfg.DBName + "?_journal_mode=WAL")
	} else {
		var dsn string
		if cfg.DatabaseURL != "" {
			dsn = cfg.DatabaseURL
		} else {
			dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
				cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
		}
		dialector = postgres.Open(dsn)
	}

	var err error
	for i := 0; i < 10; i++ {
		DB, err = gorm.Open(dialector, &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Connecting to database (attempt %d/10)... error: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to database after retries: %v", err)
	}

	log.Println("Database connection established")
}

func GetDB() *gorm.DB {
	return DB
}
