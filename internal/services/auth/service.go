package auth

import (
	"errors"
	"fmt"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/email"
	"gopickup/internal/utils"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthService struct {
	emailService email.EmailService
	config       *config.Config
}

func NewAuthService(emailService email.EmailService, cfg *config.Config) *AuthService {
	return &AuthService{
		emailService: emailService,
		config:       cfg,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=client driver vendor"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

type MeResponse struct {
	ID                uuid.UUID       `json:"id"`
	Email             string          `json:"email"`
	Role              models.UserRole `json:"role"`
	IsVerified        bool            `json:"is_verified"`
	FCMToken          string          `json:"fcm_token"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	FullName          string          `json:"full_name"`
	PhoneNumber       string          `json:"phone_number"`
	Address           string          `json:"address"`
	ProfilePictureURL string          `json:"profile_picture_url"`
	ProfilePicture    string          `json:"profile_picture"`
	IsApproved        bool            `json:"is_approved"`
}

func (s *AuthService) Register(req RegisterRequest) (string, *MeResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	var isNewUser bool = true

	// Check if user exists
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err == nil {
		if user.IsVerified {
			return "", nil, errors.New("email already registered")
		}
		// User exists but not verified
		isNewUser = false
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, err // DB error
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return "", nil, err
	}

	otp := utils.GenerateOTP()

	if isNewUser {
		user = models.User{
			ID:           uuid.New(), // Explicitly set ID
			Email:        req.Email,
			PasswordHash: hashedPassword,
			Role:         models.UserRole(req.Role),
			OTPCode:      otp,
			OTPExpiresAt: time.Now().Add(10 * time.Minute),
			CreatedAt:    time.Now(), // Explicitly set timestamps
			UpdatedAt:    time.Now(),
		}

		log.Printf("Attempting to create user: ID=%s Email=%s Role=%s", user.ID, user.Email, user.Role)

		if err := db.DB.Create(&user).Error; err != nil {
			log.Printf("DB Create Error: %v", err)
			return "", nil, err
		}
	} else {
		// Update existing unverified user
		user.PasswordHash = hashedPassword
		user.Role = models.UserRole(req.Role)
		user.OTPCode = otp
		user.OTPExpiresAt = time.Now().Add(10 * time.Minute)
		user.UpdatedAt = time.Now()

		log.Printf("Updating unverified user: ID=%s Email=%s", user.ID, user.Email)

		if err := db.DB.Save(&user).Error; err != nil {
			log.Printf("DB Update Error: %v", err)
			return "", nil, err
		}
	}

	// Generate JWT token for immediate login (to support WS connection and frontend state)
	token, err := utils.GenerateJWT(user.ID, string(user.Role), s.config.JWTSecret)
	if err != nil {
		return "", nil, err
	}

	// Send OTP email (verification is still required even if we allow login)
	go func() {
		emailSubject := "Your GoPickup Verification Code"

		emailHTML := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<meta charset="utf-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f6f9fc; margin: 0; padding: 0; color: #333333; }
					.wrapper { width: 100%%; background-color: #f6f9fc; padding: 40px 0; }
					.container { max-width: 460px; margin: 0 auto; background-color: #ffffff; border-radius: 16px; box-shadow: 0 4px 20px rgba(0,0,0,0.05); overflow: hidden; }
					.header { padding: 40px 40px 20px 40px; text-align: center; }
					.logo { font-size: 24px; font-weight: 800; color: #111111; letter-spacing: -0.5px; text-decoration: none; }
					.content { padding: 0 40px 40px 40px; text-align: center; }
					.title { font-size: 20px; font-weight: 600; margin-bottom: 16px; color: #1a1a1a; }
					.text { font-size: 15px; line-height: 1.6; color: #4e5c6e; margin-bottom: 32px; }
					.code-container { background-color: #f0f4f8; border-radius: 12px; padding: 24px; margin-bottom: 32px; letter-spacing: 8px; font-family: 'Courier New', monospace; font-size: 32px; font-weight: 700; color: #0055ff; }
					.footer { padding: 20px; text-align: center; color: #8898aa; font-size: 12px; line-height: 1.5; }
					.footer a { color: #8898aa; text-decoration: none; border-bottom: 1px solid #dce4ec; }
					.footer a:hover { color: #333; border-color: #333; }
				</style>
			</head>
			<body>
				<div class="wrapper">
					<div class="container">
						<div class="header">
							<div class="logo">GoPickup</div>
						</div>
						<div class="content">
							<div class="title">Verify your email</div>
							<p class="text">Welcome! Use the code below to complete your sign up. This code will expire in 10 minutes.</p>
							
							<div class="code-container">
								%s
							</div>
							
							<p class="text" style="margin-bottom: 0; font-size: 13px; color: #8898aa;">If you didn't request this, you can safely ignore this email.</p>
						</div>
					</div>
					<div class="footer">
						&copy; 2026 GoPickup Inc.<br>
						San Francisco, CA
					</div>
				</div>
			</body>
			</html>
		`, otp)

		emailText := fmt.Sprintf("Welcome to GoPickup!\n\nYour verification code is: %s\n\nThis code will expire in 10 minutes.", otp)

		err := s.emailService.SendEmail(
			user.Email,
			emailSubject,
			emailHTML,
			emailText,
		)
		if err != nil {
			log.Printf("Failed to send OTP email to %s: %v", user.Email, err)
		}
	}()

	fcmToken := ""
	if user.FCMToken != nil {
		fcmToken = *user.FCMToken
	}

	return token, &MeResponse{
		ID:                user.ID,
		Email:             user.Email,
		Role:              user.Role,
		IsVerified:        user.IsVerified,
		FCMToken:          fcmToken,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
		FullName:          "", // No name at registration
		PhoneNumber:       "",
		Address:           "",
		ProfilePictureURL: "",
		ProfilePicture:    "",
		IsApproved:        false,
	}, nil
}

func (s *AuthService) sendWelcomeEmail(email string) {
	subject := "Welcome to GoPickup!"
	htmlBody := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f6f9fc; margin: 0; padding: 0; color: #333333; }
				.wrapper { width: 100%%; background-color: #f6f9fc; padding: 40px 0; }
				.container { max-width: 460px; margin: 0 auto; background-color: #ffffff; border-radius: 16px; box-shadow: 0 4px 20px rgba(0,0,0,0.05); overflow: hidden; }
				.header { background-color: #0055ff; padding: 40px; text-align: center; }
				.logo { font-size: 24px; font-weight: 800; color: #ffffff; letter-spacing: -0.5px; text-decoration: none; }
				.content { padding: 40px; text-align: center; }
				.title { font-size: 24px; font-weight: 700; margin-bottom: 16px; color: #1a1a1a; }
				.text { font-size: 16px; line-height: 1.6; color: #4e5c6e; margin-bottom: 32px; }
				.button { display: inline-block; background-color: #0055ff; color: #ffffff; font-weight: 600; padding: 12px 32px; border-radius: 8px; text-decoration: none; transition: background-color 0.2s; }
				.button:hover { background-color: #0044cc; }
				.footer { padding: 20px; text-align: center; color: #8898aa; font-size: 12px; line-height: 1.5; }
			</style>
		</head>
		<body>
			<div class="wrapper">
				<div class="container">
					<div class="header">
						<div class="logo">GoPickup</div>
					</div>
					<div class="content">
						<div class="title">Welcome Aboard! 🎉</div>
						<p class="text">We're thrilled to have you join GoPickup. You've taken the first step towards smarter, more efficient logistics.</p>
						<p class="text">Your account is ready to go. Explore the marketplace, connect with drivers, or start managing your fleet today.</p>
						<a href="https://main.gopickup.com.ng" class="button">Go to Dashboard</a>
					</div>
				</div>
				<div class="footer">
					&copy; 2026 GoPickup Inc.<br>
					Making logistics simple.
				</div>
			</div>
		</body>
		</html>
	`
	textBody := "Welcome to GoPickup! We're thrilled to have you join us. Your account is ready to go."

	if err := s.emailService.SendEmail(email, subject, htmlBody, textBody); err != nil {
		log.Printf("Failed to send welcome email to %s: %v", email, err)
	}
}

func (s *AuthService) Me(userID uuid.UUID) (*MeResponse, error) {
	var user models.User
	// Load user with relations to populate profile fields
	if err := db.DB.Preload("ClientProfile").
		Preload("DriverProfile").
		Preload("VendorProfile").
		First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	fcmToken := ""
	if user.FCMToken != nil {
		fcmToken = *user.FCMToken
	}

	fullName := ""
	phoneNumber := ""
	address := ""
	profilePictureURL := ""
	isApproved := false

	switch user.Role {
	case models.RoleClient:
		if user.ClientProfile != nil {
			fullName = user.ClientProfile.FullName
			phoneNumber = user.ClientProfile.PhoneNumber
			address = user.ClientProfile.Address
			if user.ClientProfile.ProfilePictureURL != nil {
				profilePictureURL = *user.ClientProfile.ProfilePictureURL
			}
			isApproved = true // Clients are always approved
		}
	case models.RoleDriver:
		if user.DriverProfile != nil {
			fullName = user.DriverProfile.FullName
			phoneNumber = user.DriverProfile.PhoneNumber
			if user.DriverProfile.ProfilePictureURL != nil {
				profilePictureURL = *user.DriverProfile.ProfilePictureURL
			}
			isApproved = user.DriverProfile.IsApproved
		}
	case models.RoleVendor:
		if user.VendorProfile != nil {
			fullName = user.VendorProfile.StoreName
			phoneNumber = user.VendorProfile.PhoneNumber
			address = user.VendorProfile.Address
			if user.VendorProfile.StoreBannerURL != nil {
				profilePictureURL = *user.VendorProfile.StoreBannerURL
			}
			isApproved = user.VendorProfile.IsApproved
		}
	case models.RoleAdmin:
		fullName = "Admin"
		isApproved = true
	}

	return &MeResponse{
		ID:                user.ID,
		Email:             user.Email,
		Role:              user.Role,
		IsVerified:        user.IsVerified,
		FCMToken:          fcmToken,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
		FullName:          fullName,
		PhoneNumber:       phoneNumber,
		Address:           address,
		ProfilePictureURL: profilePictureURL,
		ProfilePicture:    profilePictureURL,
		IsApproved:        isApproved,
	}, nil
}

func (s *AuthService) Login(req LoginRequest) (string, *MeResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Login failed: User not found for email: %s", req.Email)
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, err
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		log.Printf("Login failed: Password mismatch for user: %s", req.Email)
		return "", nil, errors.New("invalid credentials")
	}

	// Allow unverified users to login to prevent frontend crash
	// if !user.IsVerified {
	// 	return "", nil, errors.New("account not verified. please verify your email")
	// }

	token, err := utils.GenerateJWT(user.ID, string(user.Role), s.config.JWTSecret)
	if err != nil {
		return "", nil, err
	}

	me, err := s.Me(user.ID)
	if err != nil {
		return "", nil, err
	}

	return token, me, nil
}

func (s *AuthService) AdminLogin(req LoginRequest) (string, *MeResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Admin Login failed: User not found for email: %s", req.Email)
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, err
	}

	if user.Role != models.RoleAdmin {
		log.Printf("Admin Login failed: User %s is not an admin", req.Email)
		return "", nil, errors.New("unauthorized: admin privileges required")
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		log.Printf("Admin Login failed: Password mismatch for user: %s", req.Email)
		return "", nil, errors.New("invalid credentials")
	}

	token, err := utils.GenerateJWT(user.ID, string(user.Role), s.config.JWTSecret)
	if err != nil {
		return "", nil, err
	}

	me, err := s.Me(user.ID)
	if err != nil {
		return "", nil, err
	}

	return token, me, nil
}

func (s *AuthService) VerifyOTP(req VerifyOTPRequest) (string, *MeResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.OTP = strings.ReplaceAll(req.OTP, " ", "") // Remove all spaces from OTP input

	var user models.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return "", nil, errors.New("user not found")
	}

	if user.IsVerified {
		return "", nil, errors.New("user already verified")
	}

	// Compare OTP (ensure case-insensitive for robustness, though usually numeric)
	if user.OTPCode != req.OTP {
		log.Printf("OTP Verification Failed for %s: Expected '%s', Got '%s'", req.Email, user.OTPCode, req.OTP)
		return "", nil, errors.New("invalid OTP")
	}

	if time.Now().After(user.OTPExpiresAt) {
		return "", nil, errors.New("OTP expired")
	}

	// Verify user
	user.IsVerified = true
	user.OTPCode = "" // Clear OTP
	if err := db.DB.Save(&user).Error; err != nil {
		return "", nil, err
	}

	// Send Welcome Email
	go s.sendWelcomeEmail(user.Email)

	// Generate token
	token, err := utils.GenerateJWT(user.ID, string(user.Role), s.config.JWTSecret)
	if err != nil {
		return "", nil, err
	}

	me, err := s.Me(user.ID)
	if err != nil {
		return "", nil, err
	}

	return token, me, nil
}
