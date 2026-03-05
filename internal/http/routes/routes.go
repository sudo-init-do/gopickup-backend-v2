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
	"gopickup/internal/services/audit"
	orderHandler "gopickup/internal/http/handlers/order"
	driverHandler "gopickup/internal/http/handlers/driver"
	"gopickup/internal/services/order"
	"gopickup/internal/services/profile"
	"gopickup/internal/services/product"
	"gopickup/internal/services/driver"
	"gopickup/internal/services/notification"
	wsHandler "gopickup/internal/http/handlers/websocket"
	chatHandler "gopickup/internal/http/handlers/chat"
	"gopickup/internal/db"
	"gopickup/internal/services/chat"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.New() // Use New() to skip default logger/recovery as we add custom ones
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.CORSMiddleware())

	// Services
	auditService := audit.NewAuditService(db.GetDB())
	emailService := email.NewPlunkService(cfg)
	authService := auth.NewAuthService(emailService, cfg)
	profileService := profile.NewProfileService(emailService)
	productService := product.NewProductService(auditService)
	orderService := order.NewOrderService(auditService)
	driverService := driver.NewDriverService(auditService)
	notifService := notification.NewNotificationService(db.GetDB())
	chatService := chat.NewChatService(db.GetDB(), notifService, auditService)

	// Handlers
	authH := authHandler.NewAuthHandler(authService)
	profileH := profileHandler.NewProfileHandler(profileService)
	productH := productHandler.NewProductHandler(productService)
	orderH := orderHandler.NewOrderHandler(orderService)
	driverH := driverHandler.NewDriverHandler(driverService)
	wsH := wsHandler.NewHandler(notifService, orderService, driverService, db.GetDB(), cfg)
	chatH := chatHandler.NewHandler(chatService)

	// Public Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)
		api.GET("/ready", func(c *gin.Context) {
			sqlDB, err := db.GetDB().DB()
			if err != nil {
				c.JSON(503, gin.H{"status": "down", "error": "db_connection_error"})
				return
			}
			if err := sqlDB.Ping(); err != nil {
				c.JSON(503, gin.H{"status": "down", "error": "db_ping_failed"})
				return
			}
			c.JSON(200, gin.H{"status": "up"})
		})
		api.GET("/ws", wsH.HandleConnection) // WebSocket Endpoint

		api.GET("/products", productH.ListProducts)
		api.GET("/products/:id", productH.GetProduct)
		api.GET("/vendors", productH.ListVendors)

		authGroup := api.Group("/auth")
		authGroup.Use(middleware.RateLimitMiddleware(rate.Limit(5), 10))
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/verify-otp", authH.VerifyOTP)
		}

		// Protected Routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			// Chat Routes
			chatGroup := protected.Group("/chats")
			{
				chatGroup.POST("/initiate", chatH.InitiateChat)
				chatGroup.GET("", chatH.GetChats)
				chatGroup.GET("/:id/messages", chatH.GetMessages)
				chatGroup.PATCH("/:id/read", chatH.MarkRead)
			}

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
			}

			// Notification Routes
			notifGroup := protected.Group("/notifications")
			{
				notifGroup.PUT("/fcm-token", wsH.UpdateFCMToken)
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
