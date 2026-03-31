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
	// 1. Create a Developer Admin Account
	createDevAccount(db, "developer@gopickup.com.ng", "DevSecure123!", models.RoleAdmin, nil)

	// 2. Create a Developer Vendor Account (so they don't have to fight frontend profile creation)
	vendorProfile := func(userID uuid.UUID) {
		profile := models.VendorProfile{
			UserID:         userID,
			StoreName:      "Developer Test Store",
			PhoneNumber:    "08000000001",
			BusinessType:   "Software & Testing",
			Address:        "123 Dev Lane, Lagos",
			IsApproved:     true,
		}
		db.FirstOrCreate(&profile, models.VendorProfile{UserID: userID})
	}
	createDevAccount(db, "vendor.dev@gopickup.com.ng", "DevSecure123!", models.RoleVendor, vendorProfile)
	// 3. Create a Developer Client Account
	clientProfile := func(userID uuid.UUID) {
		profile := models.ClientProfile{
			UserID:      userID,
			FullName:    "Developer Test Client",
			PhoneNumber: "08000000002",
			Address:     "123 client Lane, Lagos",
		}
		db.FirstOrCreate(&profile, models.ClientProfile{UserID: userID})
	}
	createDevAccount(db, "client.dev@gopickup.com.ng", "DevSecure123!", models.RoleClient, clientProfile)
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
