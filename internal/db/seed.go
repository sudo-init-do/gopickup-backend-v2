package db

import (
	"gopickup/internal/models"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedDeveloperAccounts ensures an admin account exists.
//
// It NEVER resets an existing account's password or role (that would let a
// committed default password override an operator-changed one on every boot).
// Credentials come from ADMIN_EMAIL/ADMIN_PASSWORD. In production those env
// vars are required; outside production a local-only default is used so dev
// setups still work out of the box.
func SeedDeveloperAccounts(db *gorm.DB) {
	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")

	if email == "" || password == "" {
		if os.Getenv("APP_ENV") == "production" {
			log.Println("Admin seeding skipped: set ADMIN_EMAIL and ADMIN_PASSWORD to seed an admin in production.")
			return
		}
		// Local development fallback only.
		email = "admin@gopickup.local"
		password = "ChangeMe123!"
	}

	createDevAccount(db, email, password, models.RoleAdmin, nil)
}

func createDevAccount(db *gorm.DB, email, password string, role models.UserRole, profileSetup func(uuid.UUID)) {
	var user models.User
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err := db.Where("email = ?", email).First(&user).Error; err == nil {
		// Account already exists — leave its password and role untouched.
		log.Printf("Admin account already exists, leaving credentials unchanged: %s", email)
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


