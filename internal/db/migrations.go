package db

import (
	"log"

	"gorm.io/gorm"
)

func RunCustomMigrations(db *gorm.DB) {
	if db.Dialector.Name() == "postgres" {
		log.Println("Running PostgreSQL specific migrations...")
		fixUserPasswordSchema(db)
		createProductSearchIndex(db)
	}
}

func fixUserPasswordSchema(db *gorm.DB) {
	// Check if 'password' column exists and make it nullable to avoid conflicts with 'password_hash'
	var count int64
	db.Raw("SELECT count(*) FROM information_schema.columns WHERE table_name='users' AND column_name='password'").Scan(&count)
	if count > 0 {
		log.Println("Fixing 'users' table schema: Dropping NOT NULL from 'password' column...")
		// We use EXEC to run raw SQL. We ignore error if it's already nullable or other minor issues, 
		// but logging it is good.
		err := db.Exec(`ALTER TABLE users ALTER COLUMN password DROP NOT NULL`).Error
		if err != nil {
			log.Printf("Failed to alter 'password' column (might already be nullable): %v", err)
		} else {
			log.Println("Successfully made 'password' column nullable.")
		}
	}
}

func createProductSearchIndex(db *gorm.DB) {
	// Check if index exists
	// This is a simple check, in a real migration tool we would track versions
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
