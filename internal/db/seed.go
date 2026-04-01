package db

import (
	"context"
	"gopickup/internal/models"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedDeveloperAccounts runs automatically on startup to ensure a dev/admin account exists.
func SeedDeveloperAccounts(db *gorm.DB) {
	createDevAccount(db, "admin@gopickup.com.ng", "Admin@2026!", models.RoleAdmin, nil)
}

func createDevAccount(db *gorm.DB, email, password string, role models.UserRole, profileSetup func(uuid.UUID)) {
	var user models.User
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err := db.Where("email = ?", email).First(&user).Error; err == nil {
		// Account exists. Force upgrade role and reset password to ensure it works.
		user.Role = role
		user.PasswordHash = string(hashedPassword)
		db.Save(&user)

		log.Printf("Forced admin role and reset password for: %s", email)

		if profileSetup != nil {
			profileSetup(user.ID)
		}
		return
	}

	userID := uuid.New()

	user = models.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		IsVerified:   true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(&user).Error; err != nil {
		log.Printf("Failed to seed dev account %s: %v", email, err)
		return
	}

	log.Printf("Seeded developer account: %s", email)

	if profileSetup != nil {
		profileSetup(userID)
	}
}

// oba look at this
// WipeMarketplaceData erases all marketplace data (products, orders, items) and flushes the cache.
func WipeMarketplaceData(db *gorm.DB) {
	log.Println("🚨 WIPE_MARKETPLACE=true detected! Starting data erasure...")

	// 1. Flush Redis to clear cached product lists
	if RedisClient != nil {
		if err := RedisClient.FlushAll(context.Background()).Err(); err != nil {
			log.Printf("⚠️  Failed to flush Redis: %v", err)
		} else {
			log.Println("🧹 Redis cache cleared.")
		}
	}

	// 2. Hard delete records (raw SQL to bypass mass-deletion protection)
	queries := []string{
		"DELETE FROM order_items",
		"DELETE FROM orders",
		"DELETE FROM products",
		"DELETE FROM bids",
	}

	for _, q := range queries {
		if err := db.Exec(q).Error; err != nil {
			log.Printf("❌ Failed to execute wipe query [%s]: %v", q, err)
		}
	}

	log.Println("✅ Marketplace data and cache completely wiped.")
}
