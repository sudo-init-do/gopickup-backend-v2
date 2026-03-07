package db

import (
	"gopickup/internal/models"
	"log"

	"gorm.io/gorm"
)

func RunCustomMigrations(db *gorm.DB) {
	if db.Dialector.Name() == "postgres" {
		log.Println("Running PostgreSQL specific migrations...")
		fixUserPasswordSchema(db)
		fixOrderSchema(db)
		fixMessageSchema(db)
		createProductSearchIndex(db)
	}
}

func fixOrderSchema(db *gorm.DB) {
	// Check if 'orders' table exists and if 'id' is NOT uuid
	var dataType string
	db.Raw("SELECT data_type FROM information_schema.columns WHERE table_name='orders' AND column_name='id'").Scan(&dataType)

	if dataType != "" && dataType != "uuid" {
		log.Printf("Detected incorrect schema for 'orders' table (ID type is %s). Recreating tables...", dataType)
		
		if err := db.Exec(`DROP TABLE IF EXISTS order_items CASCADE`).Error; err != nil {
			log.Printf("Failed to drop order_items table: %v", err)
		}
		if err := db.Exec(`DROP TABLE IF EXISTS orders CASCADE`).Error; err != nil {
			log.Printf("Failed to drop orders table: %v", err)
		}

		log.Println("Recreating orders tables with correct schema...")
		if err := db.AutoMigrate(&models.Order{}, &models.OrderItem{}); err != nil {
			log.Printf("Failed to recreate tables: %v", err)
		} else {
			log.Println("Successfully recreated 'orders' and 'order_items' tables with UUIDs.")
		}
		return
	}

	// Check if 'total_amount' column exists and make it nullable (legacy support)
	var count int64
	db.Raw("SELECT count(*) FROM information_schema.columns WHERE table_name='orders' AND column_name='total_amount'").Scan(&count)
	if count > 0 {
		log.Println("Fixing 'orders' table schema: Dropping NOT NULL from 'total_amount' column...")
		err := db.Exec(`ALTER TABLE orders ALTER COLUMN total_amount DROP NOT NULL`).Error
		if err != nil {
			log.Printf("Failed to alter 'total_amount' column: %v", err)
		} else {
			log.Println("Successfully made 'total_amount' column nullable.")
		}
	}
}

func fixMessageSchema(db *gorm.DB) {
	// Check if 'messages' table exists and if 'id' is NOT uuid (i.e., bigint/integer)
	var dataType string
	db.Raw("SELECT data_type FROM information_schema.columns WHERE table_name='messages' AND column_name='id'").Scan(&dataType)
	
	if dataType != "" && dataType != "uuid" {
		log.Printf("Detected incorrect schema for 'messages' table (ID type is %s). Recreating tables...", dataType)
		
		// Drop tables with cascade to handle foreign keys
		if err := db.Exec(`DROP TABLE IF EXISTS messages CASCADE`).Error; err != nil {
			log.Printf("Failed to drop messages table: %v", err)
		}
		if err := db.Exec(`DROP TABLE IF EXISTS chat_participants CASCADE`).Error; err != nil {
			log.Printf("Failed to drop chat_participants table: %v", err)
		}
		if err := db.Exec(`DROP TABLE IF EXISTS chats CASCADE`).Error; err != nil {
			log.Printf("Failed to drop chats table: %v", err)
		}

		// Re-run AutoMigrate for these tables
		log.Println("Recreating tables with correct schema...")
		if err := db.AutoMigrate(&models.Chat{}, &models.Message{}); err != nil {
			log.Printf("Failed to recreate tables: %v", err)
		} else {
			log.Println("Successfully recreated 'chats' and 'messages' tables with UUIDs.")
		}
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
