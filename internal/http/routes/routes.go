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
	orderHandler "gopickup/internal/http/handlers/order"
	driverHandler "gopickup/internal/http/handlers/driver"
	"gopickup/internal/services/order"
	"gopickup/internal/services/profile"
	"gopickup/internal/services/product"
	"gopickup/internal/services/driver"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// Services
	emailService := email.NewPlunkService(cfg)
	authService := auth.NewAuthService(emailService, cfg)
	profileService := profile.NewProfileService(emailService)
	productService := product.NewProductService()
	orderService := order.NewOrderService()
	driverService := driver.NewDriverService()

	// Handlers
	authH := authHandler.NewAuthHandler(authService)
	profileH := profileHandler.NewProfileHandler(profileService)
	productH := productHandler.NewProductHandler(productService)
	orderH := orderHandler.NewOrderHandler(orderService)
	driverH := driverHandler.NewDriverHandler(driverService)

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
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			protected.GET("/auth/me", authH.Me)
			protected.GET("/orders", orderH.ListOrders)
			protected.GET("/orders/:id", orderH.GetOrder)

			// Profile Routes
			profileGroup := protected.Group("/profile")
			{
				profileGroup.POST("/client", middleware.RoleMiddleware(string(models.RoleClient)), profileH.CreateClient)
				profileGroup.POST("/driver", middleware.RoleMiddleware(string(models.RoleDriver)), profileH.CreateDriver)
				profileGroup.POST("/vendor", middleware.RoleMiddleware(string(models.RoleVendor)), profileH.CreateVendor)
				profileGroup.PUT("/", profileH.UpdateProfile)
			}

			// Client Routes
			clientGroup := protected.Group("/orders")
			clientGroup.Use(middleware.RoleMiddleware(string(models.RoleClient)))
			{
				clientGroup.POST("/checkout", orderH.Checkout)
				clientGroup.GET("/:id/bids", orderH.GetBids)
				clientGroup.POST("/:id/bids/:bid_id/accept", orderH.AcceptBid)
			}

			// Driver Routes
			driverGroup := protected.Group("/driver")
			driverGroup.Use(middleware.RoleMiddleware(string(models.RoleDriver)))
			{
				driverGroup.PATCH("/location", driverH.UpdateLocation)
			}

			jobsGroup := protected.Group("/jobs")
			jobsGroup.Use(middleware.RoleMiddleware(string(models.RoleDriver)))
			{
				jobsGroup.GET("/available", driverH.GetAvailableJobs)
				jobsGroup.POST("/:order_id/bid", driverH.PlaceBid)
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
				vendorGroup.PATCH("/orders/:id/status", orderH.VendorUpdateStatus)
				vendorGroup.PATCH("/orders/:id/ready", orderH.VendorMarkReady)
			}
		}
	}

	return r
}
