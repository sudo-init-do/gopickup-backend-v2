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
	createAdmin := flag.Bool("admin", false, "Create an admin user")
	verifyUser := flag.String("verify", "", "Email of user to verify (mark verified=true)")
	emailFlag := flag.String("email", "admin@gopickup.com", "Email for admin user")
	passwordFlag := flag.String("password", "AdminPassword123!", "Password for admin user")
	flag.Parse()

	// 2. Load Config & Connect DB
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Safety check: Don't run in production unless explicitly forced (maybe add a force flag, but for now just warn)
	if cfg.AppEnv == "production" {
		log.Println("WARNING: Running seed tool in PRODUCTION environment.")
		// In a real scenario, we might want to block this or require a confirmation.
		// For now, we'll proceed but logging is critical.
	}

	db.Connect(cfg)

	// 3. Execute requested action
	if *createAdmin {
		createAdminUser(*emailFlag, *passwordFlag)
	}

	if *verifyUser != "" {
		verifyUserByEmail(*verifyUser)
	}

	if !*createAdmin && *verifyUser == "" {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/seed/main.go -admin -email=... -password=...")
		fmt.Println("  go run cmd/seed/main.go -verify=user@example.com")
		os.Exit(1)
	}
}

func createAdminUser(email, password string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	admin := models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         models.RoleAdmin,
		IsVerified:   true,
	}

	if err := db.GetDB().Create(&admin).Error; err != nil {
		log.Printf("Failed to create admin user (maybe exists?): %v", err)
	} else {
		log.Printf("Admin user created: %s", email)
	}
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
