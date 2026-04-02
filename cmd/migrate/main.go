package main

import (
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"log"
	"os"
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

	// Optional Purge
	if os.Getenv("PURGE_DATABASE") == "true" {
		log.Println("PURGE_DATABASE=true detected! Wiping all data...")
		db.GetDB().Exec("DROP TABLE IF EXISTS audit_logs, load_bids, loads, transactions, wallets, messages, chats, order_items, orders, products, vendor_profiles, driver_profiles, client_profiles, users CASCADE")
		log.Println("Database wiped.")
	}

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
