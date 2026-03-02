package main

import (
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/http/routes"
	"gopickup/internal/models"
	"log"
)

func main() {
	log.Println("Starting application...")

	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Database
	db.Connect(cfg)

	// Auto Migrate
	log.Println("Running migrations...")
	if err := db.GetDB().AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Setup Router
	r := routes.SetupRouter(cfg)

	log.Printf("Starting server on port %s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
