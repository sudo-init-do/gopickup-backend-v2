package main

import (
	"context"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"log"
	"os"
)

func main() {
	log.Println("Starting database migration...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db.Connect(cfg)
	db.InitRedis(cfg)

	// NUCLEAR PURGE: If PURGE_DATABASE is true, wipe everything dynamically
	if os.Getenv("PURGE_DATABASE") == "true" {
		log.Println("⚠️ PURGE_DATABASE=true detected! Initiating Nuclear Dynamo Wipe...")
		
		// 1. Dynamic Table Discovery & Destruction
		database := db.GetDB()
		migrator := database.Migrator()
		
		tables, err := migrator.GetTables()
		if err != nil {
			log.Printf("❌ Failed to list tables: %v", err)
		} else {
			log.Printf("🛰️ Found %d tables. Commencing destruction...", len(tables))
			for _, table := range tables {
				log.Printf("🥊 Dropping table: %s", table)
				if err := database.Exec("DROP TABLE IF EXISTS " + table + " CASCADE").Error; err != nil {
					log.Printf("⚠️ Warning: Failed to drop table %s: %v (trying non-cascade)", table, err)
					// Fallback for DBs that don't support CASCADE (like SQLite)
					database.Exec("DROP TABLE IF EXISTS " + table)
				}
			}
			log.Println("🧼 All tables dropped successfully.")
		}

		// 2. Redis Flush
		redisClient := db.GetRedis()
		if redisClient != nil {
			log.Println("🔥 Flushing Redis cache...")
			if err := redisClient.FlushAll(context.Background()).Err(); err != nil {
				log.Printf("⚠️ Warning: Redis flush failed: %v", err)
			} else {
				log.Println("🌊 Redis flush complete.")
			}
		}

		log.Println("✅ Nuclear Dynamo Wipe complete. Ready for clean AutoMigrate.")
	}

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
