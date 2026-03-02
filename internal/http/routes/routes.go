package routes

import (
	"gopickup/internal/config"
	"gopickup/internal/http/handlers"
	authHandler "gopickup/internal/http/handlers/auth"
	"gopickup/internal/http/middleware"
	"gopickup/internal/services/auth"
	"gopickup/internal/services/email"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// Services
	emailService := email.NewPlunkService(cfg)
	authService := auth.NewAuthService(emailService, cfg)

	// Handlers
	authH := authHandler.NewAuthHandler(authService)

	// Public Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)

		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/verify-otp", authH.VerifyOTP)

			// Protected Routes
			protected := authGroup.Group("/")
			protected.Use(middleware.AuthMiddleware(cfg))
			{
				protected.GET("/me", authH.Me)
			}
		}
	}

	return r
}
