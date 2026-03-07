package main

import (
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"log"
)

func main() {
	log.Println("Starting database migration...")

	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Database
	db.Connect(cfg)

	// Run Migrations
	log.Println("Running auto-migration...")
	if err := db.GetDB().AutoMigrate(
		&models.User{},
		&models.ClientProfile{},
		&models.DriverProfile{},
		&models.VendorProfile{},
		&models.Product{},
		&models.Order{},
		&models.OrderItem{},
		&models.Bid{},
		&models.Chat{},
		&models.Message{},
		&models.Wallet{},
		&models.Transaction{},
		&models.AuditLog{}, // Added AuditLog as it was likely missed in main.go or I should check if it exists
	); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully.")
	
	// Run custom migrations/fixes
	db.RunCustomMigrations(db.GetDB())
}
