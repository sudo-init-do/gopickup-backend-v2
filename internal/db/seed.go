package db

import (
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
func WipeMarketplaceData(db *gorm.DB) {
	log.Println("🚨 WIPE_MARKETPLACE=true detected! Erasing all Marketplace Data (Products, Orders, OrderItems)...")

	// Hard delete (raw SQL to bypass Gorm's AllowGlobalUpdate protection)
	db.Exec("DELETE FROM order_items")
	db.Exec("DELETE FROM orders")
	db.Exec("DELETE FROM products")
	
	log.Println("✅ Marketplace Data completely wiped.")
}
