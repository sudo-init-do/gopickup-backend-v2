package db

import (
	"gopickup/internal/models"
	"log"

	"strings"

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
	log.Println("Checking 'orders' table schema for UUID compatibility...")
	
	// Use GORM Migrator to check column type for 'id'
	columnTypes, err := db.Migrator().ColumnTypes("orders")
	if err != nil {
		log.Printf("Failed to get column types for orders (might not exist yet): %v", err)
		return
	}

	foundID := false
	for _, ct := range columnTypes {
		name := strings.ToLower(ct.Name())
		dataType := strings.ToLower(ct.DatabaseTypeName())
		log.Printf("DEBUG: Column detected in 'orders': %s (%s)", name, dataType)

		if name == "id" {
			foundID = true
			// Postgres uuid type is usually "uuid"
			if dataType != "uuid" {
				log.Printf("Detected incorrect schema for 'orders' table (ID type is %s, expected uuid). Recreating tables...", dataType)
				recreateOrderTables(db)
				return
			}
			log.Println("Order ID column is already UUID. Checking 'order_items'...")
			break
		}
	}

	if !foundID {
		log.Println("Order ID column not found in 'orders' table. Recreating tables to ensure correct schema...")
		recreateOrderTables(db)
		return
	}

	// Also check order_items for bigint order_id
	itemColumnTypes, err := db.Migrator().ColumnTypes("order_items")
	if err == nil {
		for _, ct := range itemColumnTypes {
			name := strings.ToLower(ct.Name())
			dataType := strings.ToLower(ct.DatabaseTypeName())
			if (name == "id" || name == "order_id") && dataType != "uuid" {
				log.Printf("Detected incorrect schema for 'order_items' table (%s type is %s). Recreating tables...", name, dataType)
				recreateOrderTables(db)
				return
			}
		}
	}

	// Check if 'total_amount' column exists and make it nullable (legacy support)
	if db.Migrator().HasColumn(&models.Order{}, "total_amount") {
		log.Println("Fixing 'orders' table schema: Dropping NOT NULL from 'total_amount' column...")
		err := db.Exec(`ALTER TABLE orders ALTER COLUMN total_amount DROP NOT NULL`).Error
		if err != nil {
			log.Printf("Failed to alter 'total_amount' column: %v", err)
		} else {
			log.Println("Successfully made 'total_amount' column nullable.")
		}
	}
}

func recreateOrderTables(db *gorm.DB) {
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
}

func fixMessageSchema(db *gorm.DB) {
	log.Println("Checking 'messages' and 'chats' table schema for UUID compatibility...")
	
	// Use GORM Migrator to check column type for 'id'
	columnTypes, err := db.Migrator().ColumnTypes("messages")
	if err != nil {
		log.Printf("Failed to get column types for messages (might not exist yet): %v", err)
		return
	}

	foundID := false
	for _, ct := range columnTypes {
		name := strings.ToLower(ct.Name())
		dataType := strings.ToLower(ct.DatabaseTypeName())
		log.Printf("DEBUG: Column detected in 'messages': %s (%s)", name, dataType)

		if name == "id" {
			foundID = true
			if dataType != "uuid" {
				log.Printf("Detected incorrect schema for 'messages' table (ID type is %s). Recreating tables...", dataType)
				recreateMessageTables(db)
				return
			}
			log.Println("Message ID column is already UUID.")
			break
		}
	}

	if !foundID {
		log.Println("Message ID column not found in 'messages' table. Recreating tables...")
		recreateMessageTables(db)
		return
	}

	// Also check chat_participants for UUID
	participantColumnTypes, err := db.Migrator().ColumnTypes("chat_participants")
	if err == nil {
		for _, ct := range participantColumnTypes {
			name := strings.ToLower(ct.Name())
			dataType := strings.ToLower(ct.DatabaseTypeName())
			if (name == "chat_id" || name == "user_id") && dataType != "uuid" {
				log.Printf("Detected incorrect schema for 'chat_participants' table (%s type is %s). Recreating tables...", name, dataType)
				recreateMessageTables(db)
				return
			}
		}
	}
}

func recreateMessageTables(db *gorm.DB) {
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
