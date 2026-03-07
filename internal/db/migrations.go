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
	// Use GORM Migrator to check column type for 'id'
	// This is more reliable than raw SQL against information_schema
	columnTypes, err := db.Migrator().ColumnTypes("orders")
	if err != nil {
		log.Printf("Failed to get column types for orders (might not exist yet): %v", err)
		return
	}

	for _, ct := range columnTypes {
		if ct.Name() == "id" {
			dataType := ct.DatabaseTypeName() // e.g., "uuid", "int8", "bigint", "integer"
			log.Printf("Order ID column type detected: %s", dataType)
			
			// Postgres uuid type is usually "uuid"
			// If it's anything else (like int8, bigint, integer), we need to fix it
			if dataType != "uuid" && dataType != "UUID" {
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
				// If we recreated the table, we don't need to fix legacy columns
				return
			}
			break
		}
	}

	// Check if 'total_amount' column exists and make it nullable (legacy support)
	// We use Migrator to check existence too
	if db.Migrator().HasColumn(&models.Order{}, "total_amount") {
		log.Println("Fixing 'orders' table schema: Dropping NOT NULL from 'total_amount' column...")
		err := db.Exec(`ALTER TABLE orders ALTER COLUMN total_amount DROP NOT NULL`).Error
		if err != nil {
			// It might fail if column doesn't exist (race condition?) or other reasons
			// But we checked HasColumn. HasColumn checks the struct? No, it checks the table.
			// Wait, HasColumn first arg is dst (struct or table name).
			// If I pass &models.Order{}, GORM might look for the field in the struct?
			// Let's use table name string to be safe if GORM supports it, or just use raw SQL for this part as it worked before.
			// The previous raw SQL worked fine for this specific check.
			log.Printf("Failed to alter 'total_amount' column: %v", err)
		} else {
			log.Println("Successfully made 'total_amount' column nullable.")
		}
	} else {
		// Fallback to raw check just in case Migrator behavior on HasColumn is tricky with struct
		var count int64
		db.Raw("SELECT count(*) FROM information_schema.columns WHERE table_name='orders' AND column_name='total_amount'").Scan(&count)
		if count > 0 {
			log.Println("Fixing 'orders' table schema (fallback check): Dropping NOT NULL from 'total_amount' column...")
			db.Exec(`ALTER TABLE orders ALTER COLUMN total_amount DROP NOT NULL`)
		}
	}
}

func fixMessageSchema(db *gorm.DB) {
	// Use GORM Migrator to check column type for 'id'
	columnTypes, err := db.Migrator().ColumnTypes("messages")
	if err != nil {
		log.Printf("Failed to get column types for messages (might not exist yet): %v", err)
		return
	}

	for _, ct := range columnTypes {
		if ct.Name() == "id" {
			dataType := ct.DatabaseTypeName()
			log.Printf("Message ID column type detected: %s", dataType)
			
			if dataType != "uuid" && dataType != "UUID" {
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
				return
			}
			break
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
