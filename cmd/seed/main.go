package main

import (
	"flag"
	"fmt"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"log"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. Parse flags
	seedAll := flag.Bool("seed", false, "Seed all test accounts (Admin, Client, Driver, Vendor)")
	createAdmin := flag.Bool("admin", false, "Create an admin user only")
	verifyUser := flag.String("verify", "", "Email of user to verify (mark verified=true)")
	emailFlag := flag.String("email", "admin@gopickup.com", "Email for admin user")
	passwordFlag := flag.String("password", "AdminPassword123!", "Password for admin user")
	flag.Parse()

	// 2. Load Config & Connect DB
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Safety check
	allowSeed := os.Getenv("ALLOW_SEED")
	if cfg.AppEnv == "production" && allowSeed != "true" {
		log.Fatal("ERROR: Seeding is disabled in PRODUCTION. Set ALLOW_SEED=true to force.")
	}

	db.Connect(cfg)

	// 3. Execute requested action
	if *seedAll {
		seedAllAccounts()
	} else if *createAdmin {
		createAdminUser(*emailFlag, *passwordFlag)
	} else if *verifyUser != "" {
		verifyUserByEmail(*verifyUser)
	} else {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/seed/main.go -seed (Create all test accounts)")
		fmt.Println("  go run cmd/seed/main.go -admin -email=... -password=...")
		fmt.Println("  go run cmd/seed/main.go -verify=user@example.com")
		os.Exit(1)
	}
}

func seedAllAccounts() {
	log.Println("Seeding test accounts...")
	
	// Admin
	createAdminUser("admin@test.com", "Password123!")

	// Client
	createClientUser("client@test.com", "Password123!")

	// Driver
	createDriverUser("driver@test.com", "Password123!")

	// Vendor
	vendorID := createVendorUser("vendor@test.com", "Password123!")
	if vendorID != uuid.Nil {
		seedProducts(vendorID)
	}
	
	log.Println("Seeding complete!")
}

func seedProducts(vendorID uuid.UUID) {
	products := []models.Product{
		{
			VendorID:      vendorID,
			Name:          "Premium Headphones",
			Description:   "Noise cancelling headphones with superior sound quality",
			Price:         25000.00,
			Category:      "Electronics",
			StockQuantity: 50,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=500&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8M3x8aGVhZHBob25lc3xlbnwwfHwwfHx8MA%3D%3D",
		},
		{
			VendorID:      vendorID,
			Name:          "Organic Coffee Beans",
			Description:   "Freshly roasted organic coffee beans, 1kg pack",
			Price:         5000.00,
			Category:      "Food",
			StockQuantity: 100,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1559056199-641a0ac8b55e?w=500&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8OXx8Y29mZmVlJTIwYmVhbnN8ZW58MHx8MHx8fDA%3D",
		},
		{
			VendorID:      vendorID,
			Name:          "Running Shoes",
			Description:   "Comfortable running shoes for all terrains",
			Price:         15000.00,
			Category:      "Fashion",
			StockQuantity: 25,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=500&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8Mnx8c2hvZXN8ZW58MHx8MHx8fDA%3D",
		},
	}

	for _, p := range products {
		p.ID = uuid.New() // Ensure ID is set
		if err := db.GetDB().Create(&p).Error; err != nil {
			log.Printf("Failed to create product %s: %v", p.Name, err)
		} else {
			log.Printf("Product created: %s", p.Name)
		}
	}
}

func createUser(email, password, role string, isVerified bool) uuid.UUID {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	userID := uuid.New()
	user := models.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         models.UserRole(role),
		IsVerified:   isVerified,
	}

	if err := db.GetDB().Create(&user).Error; err != nil {
		log.Printf("Failed to create user %s (maybe exists?): %v", email, err)
		// Try to fetch existing ID if creation failed
		var existingUser models.User
		if err := db.GetDB().Where("email = ?", email).First(&existingUser).Error; err == nil {
			return existingUser.ID
		}
		return uuid.Nil
	}
	log.Printf("User created: %s (%s)", email, role)
	return userID
}

func createClientUser(email, password string) {
	uid := createUser(email, password, string(models.RoleClient), true)
	if uid == uuid.Nil { return }

	profile := models.ClientProfile{
		UserID:      uid,
		FullName:    "Test Client",
		PhoneNumber: "+1234567890",
		Address:     "123 Client St",
	}
	if err := db.GetDB().Create(&profile).Error; err != nil {
		log.Printf("Failed to create client profile: %v", err)
	} else {
		log.Printf("Client profile created for %s", email)
	}
}

func createDriverUser(email, password string) {
	uid := createUser(email, password, string(models.RoleDriver), true)
	if uid == uuid.Nil { return }

	profile := models.DriverProfile{
		UserID:          uid,
		FullName:        "Test Driver",
		PhoneNumber:     "+1987654321",
		LicenseNumber:   "DL12345678",
		VehicleType:     models.VehicleVan,
		PlateNumber:     "VAN-001",
		VehicleCapacity: 1000,
		IsApproved:      true,
	}
	if err := db.GetDB().Create(&profile).Error; err != nil {
		log.Printf("Failed to create driver profile: %v", err)
	} else {
		log.Printf("Driver profile created for %s", email)
	}
}

func createVendorUser(email, password string) uuid.UUID {
	uid := createUser(email, password, string(models.RoleVendor), true)
	if uid == uuid.Nil { return uuid.Nil }

	profile := models.VendorProfile{
		UserID:       uid,
		StoreName:    "Test Store",
		PhoneNumber:  "+1122334455",
		BusinessType: "Retail",
		Address:      "456 Market St",
		IsApproved:   true,
	}
	if err := db.GetDB().Create(&profile).Error; err != nil {
		log.Printf("Failed to create vendor profile: %v", err)
	} else {
		log.Printf("Vendor profile created for %s", email)
	}
	return uid
}

func createAdminUser(email, password string) {
	createUser(email, password, string(models.RoleAdmin), true)
}

func verifyUserByEmail(email string) {
	result := db.GetDB().Model(&models.User{}).
		Where("email = ?", email).
		Update("is_verified", true)

	if result.Error != nil {
		log.Printf("Failed to verify user: %v", result.Error)
	} else if result.RowsAffected == 0 {
		log.Printf("User not found: %s", email)
	} else {
		log.Printf("User verified: %s", email)
	}
}
