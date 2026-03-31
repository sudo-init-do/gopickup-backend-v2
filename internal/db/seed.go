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
	// Seeding disabled for production/real data mode
}

func createDevAccount(db *gorm.DB, email, password string, role models.UserRole, profileSetup func(uuid.UUID)) {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err == nil {
		// Already exists, just ensure profile is setup if needed
		if profileSetup != nil {
			profileSetup(user.ID)
		}
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
