package main

import (
	"context"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db.Connect(cfg)
	database := db.GetDB()

	log.Println("Starting database migration...")
	if err := database.AutoMigrate(
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
		&models.AuditLog{},
		&models.Wallet{},
		&models.Transaction{},
		&models.Load{},
		&models.LoadBid{},
	); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Run custom database optimizations (e.g. Postgres specific UUID fixes)
	db.RunCustomMigrations(db.GetDB())

	log.Println("Migration completed successfully.")
}
