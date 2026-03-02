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
		err := s.emailService.SendEmail(
			user.Email,
			"Verify your GoPickup account",
			fmt.Sprintf("<h1>Your OTP is: %s</h1><p>It expires in 10 minutes.</p>", otp),
			fmt.Sprintf("Your OTP is: %s. It expires in 10 minutes.", otp),
		)
		if err != nil {
			fmt.Printf("Failed to send OTP email to %s: %v\n", user.Email, err)
		}
	}()

	return nil
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
