package main

import (
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/http/routes"
	"gopickup/internal/models"
	"log"
	"os"
)

// @title GoPickup API
// @version 1.0
// @description This is the backend API for GoPickup application.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	cwd, _ := os.Getwd()
	log.Printf("Starting application... CWD: %s", cwd)

	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Database
	db.Connect(cfg)
	db.InitRedis(cfg)
	
	// Auto Migrate (Optional in prod, default true for dev convenience)
	if os.Getenv("MIGRATE_ON_START") != "false" {
		log.Println("Running migrations...")
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

		// Run custom database optimizations (e.g. indexes)
		db.RunCustomMigrations(db.GetDB())

		// Ensure developer accounts exist
		db.SeedDeveloperAccounts(db.GetDB())
	}

	// Setup Router
	r := routes.SetupRouter(cfg)

	// Trust all proxies (running behind Traefik/Dokploy)
	// This fixes the [GIN-debug] warning and ensures ClientIP() works correctly
	r.SetTrustedProxies(nil)

	log.Printf("Starting server on port %s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
