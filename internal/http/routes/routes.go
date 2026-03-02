package routes

import (
	"gopickup/internal/config"
	"gopickup/internal/http/handlers"
	authHandler "gopickup/internal/http/handlers/auth"
	profileHandler "gopickup/internal/http/handlers/profile"
	"gopickup/internal/http/middleware"
	"gopickup/internal/models"
	"gopickup/internal/services/auth"
	"gopickup/internal/services/email"
	"gopickup/internal/services/profile"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// Services
	emailService := email.NewPlunkService(cfg)
	authService := auth.NewAuthService(emailService, cfg)
	profileService := profile.NewProfileService(emailService)

	// Handlers
	authH := authHandler.NewAuthHandler(authService)
	profileH := profileHandler.NewProfileHandler(profileService)

	// Public Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)

		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/verify-otp", authH.VerifyOTP)
		}

		// Protected Routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			protected.GET("/auth/me", authH.Me)

			// Profile Routes
			profileGroup := protected.Group("/profile")
			{
				profileGroup.POST("/client", middleware.RoleMiddleware(string(models.RoleClient)), profileH.CreateClient)
				profileGroup.POST("/driver", middleware.RoleMiddleware(string(models.RoleDriver)), profileH.CreateDriver)
				profileGroup.POST("/vendor", middleware.RoleMiddleware(string(models.RoleVendor)), profileH.CreateVendor)
				profileGroup.PUT("/", profileH.UpdateProfile)
			}

			// Admin Routes
			adminGroup := protected.Group("/admin")
			adminGroup.Use(middleware.AdminMiddleware())
			{
				adminGroup.PATCH("/drivers/:user_id/approve", profileH.ApproveDriver)
				adminGroup.PATCH("/vendors/:user_id/approve", profileH.ApproveVendor)
			}
		}
	}

	return r
}
