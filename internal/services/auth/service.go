package auth

import (
	"errors"
	"fmt"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/email"
	"gopickup/internal/utils"
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
	OTP   string `json:"otp" binding:"required,len=6"`
}

type MeResponse struct {
	ID         uuid.UUID       `json:"id"`
	Email      string          `json:"email"`
	Role       models.UserRole `json:"role"`
	IsVerified bool            `json:"is_verified"`
	FCMToken   *string         `json:"fcm_token"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (s *AuthService) Register(req RegisterRequest) error {
	var existingUser models.User
	if err := db.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return errors.New("email already registered")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}

	otp := utils.GenerateOTP()
	user := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         models.UserRole(req.Role),
		OTPCode:      otp,
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := db.DB.Create(&user).Error; err != nil {
		return err
	}

	// Send OTP email
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
			fmt.Printf("Failed to send OTP email to %s: %v\n", user.Email, err)
		}
	}()

	return nil
}

func (s *AuthService) Me(userID uuid.UUID) (*MeResponse, error) {
	var user models.User
	if err := db.DB.Select(
		"id",
		"email",
		"role",
		"is_verified",
		"fcm_token",
		"created_at",
		"updated_at",
	).Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &MeResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		FCMToken:   user.FCMToken,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}, nil
}

func (s *AuthService) Login(req LoginRequest) (string, error) {
	var user models.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("invalid credentials")
		}
		return "", err
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return "", errors.New("invalid credentials")
	}

	if !user.IsVerified {
		return "", errors.New("account not verified. please verify your email")
	}

	token, err := utils.GenerateJWT(user.ID, string(user.Role), s.config.JWTSecret)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) VerifyOTP(req VerifyOTPRequest) error {
	var user models.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return errors.New("user not found")
	}

	if user.IsVerified {
		return errors.New("user already verified")
	}

	if user.OTPCode != req.OTP {
		return errors.New("invalid OTP")
	}

	if time.Now().After(user.OTPExpiresAt) {
		return errors.New("OTP expired")
	}

	// Verify user
	user.IsVerified = true
	user.OTPCode = "" // Clear OTP
	if err := db.DB.Save(&user).Error; err != nil {
		return err
	}

	return nil
}
