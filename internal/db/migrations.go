package db

import (
	"log"

	"gorm.io/gorm"
)

func RunCustomMigrations(db *gorm.DB) {
	if db.Dialector.Name() == "postgres" {
		log.Println("Running PostgreSQL specific migrations...")
		createProductSearchIndex(db)
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
