package routes

import (
	"gopickup/internal/config"
	"gopickup/internal/http/handlers"
	authHandler "gopickup/internal/http/handlers/auth"
	profileHandler "gopickup/internal/http/handlers/profile"
	productHandler "gopickup/internal/http/handlers/product"
	"gopickup/internal/http/middleware"
	"gopickup/internal/models"
	"gopickup/internal/services/auth"
	"gopickup/internal/services/email"
	"gopickup/internal/services/profile"
	"gopickup/internal/services/product"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// Services
	emailService := email.NewPlunkService(cfg)
	authService := auth.NewAuthService(emailService, cfg)
	profileService := profile.NewProfileService(emailService)
	productService := product.NewProductService()

	// Handlers
	authH := authHandler.NewAuthHandler(authService)
	profileH := profileHandler.NewProfileHandler(profileService)
	productH := productHandler.NewProductHandler(productService)

	// Public Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)
		api.GET("/products", productH.ListProducts)
		api.GET("/products/:id", productH.GetProduct)
		api.GET("/vendors", productH.ListVendors)

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

			// Vendor Routes
			vendorGroup := protected.Group("/vendor")
			vendorGroup.Use(middleware.RoleMiddleware(string(models.RoleVendor)))
			{
				vendorGroup.POST("/products", productH.CreateProduct)
				vendorGroup.PUT("/products/:id", productH.UpdateProduct)
				vendorGroup.DELETE("/products/:id", productH.DeleteProduct)
				vendorGroup.GET("/dashboard", productH.GetVendorDashboard)
			}
		}
	}

	return r
}
