package db

import (
	"gopickup/internal/models"
	"log"

	"gorm.io/gorm"
)

// RunCustomMigrations applies non-destructive, PostgreSQL-specific schema
// adjustments and indexes.
//
// IMPORTANT: This must NEVER drop tables or delete data. AutoMigrate (run
// before this) handles adding new tables/columns. Any change that would
// destroy data must be a deliberate, separately-run migration — not something
// that fires automatically on every boot.
func RunCustomMigrations(db *gorm.DB) {
	if db.Dialector.Name() == "postgres" {
		log.Println("Running PostgreSQL specific migrations...")
		fixUserPasswordSchema(db)
		fixOrderSchema(db)
		createProductSearchIndex(db)
		dropUniquePhoneConstraints(db)
	}
}

func dropUniquePhoneConstraints(db *gorm.DB) {
	log.Println("Dropping unique constraints on phone numbers to allow easier testing...")

	// Drops indexes only (no data is removed).
	if db.Migrator().HasIndex(&models.VendorProfile{}, "uni_vendor_profiles_phone_number") {
		db.Migrator().DropIndex(&models.VendorProfile{}, "uni_vendor_profiles_phone_number")
	}
	if db.Migrator().HasIndex(&models.DriverProfile{}, "uni_driver_profiles_phone_number") {
		db.Migrator().DropIndex(&models.DriverProfile{}, "uni_driver_profiles_phone_number")
	}
	if db.Migrator().HasIndex(&models.ClientProfile{}, "uni_client_profiles_phone_number") {
		db.Migrator().DropIndex(&models.ClientProfile{}, "uni_client_profiles_phone_number")
	}
}

// fixOrderSchema applies only non-destructive fixes to the orders table.
func fixOrderSchema(db *gorm.DB) {
	if !db.Migrator().HasTable("orders") {
		log.Println("'orders' table does not exist yet; skipping schema check.")
		return
	}

	// Legacy support: make the old 'total_amount' column nullable if present.
	if db.Migrator().HasColumn(&models.Order{}, "total_amount") {
		log.Println("Ensuring 'orders.total_amount' is nullable...")
		if err := db.Exec(`ALTER TABLE orders ALTER COLUMN total_amount DROP NOT NULL`).Error; err != nil {
			log.Printf("Failed to alter 'total_amount' column (might already be nullable): %v", err)
		}
	}
}

func fixUserPasswordSchema(db *gorm.DB) {
	// Make the legacy 'password' column nullable to avoid conflicts with 'password_hash'.
	var count int64
	db.Raw("SELECT count(*) FROM information_schema.columns WHERE table_name='users' AND column_name='password'").Scan(&count)
	if count > 0 {
		log.Println("Fixing 'users' table schema: Dropping NOT NULL from 'password' column...")
		if err := db.Exec(`ALTER TABLE users ALTER COLUMN password DROP NOT NULL`).Error; err != nil {
			log.Printf("Failed to alter 'password' column (might already be nullable): %v", err)
		}
	}
}

func createProductSearchIndex(db *gorm.DB) {
	var count int64
	db.Raw("SELECT count(*) FROM pg_indexes WHERE indexname = 'idx_products_fts'").Scan(&count)
	if count == 0 {
		log.Println("Creating Full Text Search index for products...")
		// Create GIN index on name and description
		err := db.Exec(`
			CREATE INDEX idx_products_fts ON products
			USING gin(to_tsvector('english', name || ' ' || coalesce(description, '')));
		`).Error
		if err != nil {
			log.Printf("Failed to create FTS index: %v", err)
		} else {
			log.Println("FTS index created successfully")
		}
	}
}
