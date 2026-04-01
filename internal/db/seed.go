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

func WipeMarketplaceData(db *gorm.DB) {
	log.Println("🚨 WIPE_MARKETPLACE=true detected! Erasing all Marketplace Data (Products, Orders, OrderItems)...")

	// Hard delete (truncate equivalent)
	db.Unscoped().Where("1=1").Delete(&models.OrderItem{})
	db.Unscoped().Where("1=1").Delete(&models.Order{})
	db.Unscoped().Where("1=1").Delete(&models.Product{})
	
	// Optionally clear carts from redis by grabbing all keys if needed, but not strictly required.
	
	log.Println("✅ Marketplace Data completely wiped.")
}
