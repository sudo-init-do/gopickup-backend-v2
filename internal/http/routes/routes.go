package routes

import (
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/http/handlers"
	authHandler "gopickup/internal/http/handlers/auth"
	chatHandler "gopickup/internal/http/handlers/chat"
	driverHandler "gopickup/internal/http/handlers/driver"
	orderHandler "gopickup/internal/http/handlers/order"
	productHandler "gopickup/internal/http/handlers/product"
	profileHandler "gopickup/internal/http/handlers/profile"
	uploadHandler "gopickup/internal/http/handlers/upload"
	walletHandler "gopickup/internal/http/handlers/wallet"
	wsHandler "gopickup/internal/http/handlers/websocket"
	"gopickup/internal/http/middleware"
	"gopickup/internal/models"
	"gopickup/internal/services/audit"
	"gopickup/internal/services/auth"
	"gopickup/internal/services/chat"
	"gopickup/internal/services/driver"
	"gopickup/internal/services/email"
	"gopickup/internal/services/notification"
	"gopickup/internal/services/order"
	"gopickup/internal/services/product"
	"gopickup/internal/services/profile"
	"gopickup/internal/services/load"
	"gopickup/internal/services/wallet"
	"gopickup/internal/services/admin"

	loadHandler "gopickup/internal/http/handlers/load"
	adminHandler "gopickup/internal/http/handlers/admin"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"os"
	"path/filepath"
	"log"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.New() // Use New() to skip default logger/recovery as we add custom ones
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.BodySizeLimitMiddleware(10 * 1024 * 1024)) // 10MB limit
	r.Use(middleware.CORSMiddleware(cfg))

	// Services
	auditService := audit.NewAuditService(db.GetDB())
	emailService := email.NewPlunkService(cfg)
	authService := auth.NewAuthService(emailService, cfg)
	profileService := profile.NewProfileService(emailService, db.GetRedis())
	productService := product.NewProductService(auditService, db.GetRedis(), cfg)
	orderService := order.NewOrderService(auditService, cfg)
	driverService := driver.NewDriverService(auditService, cfg)
	notifService := notification.NewNotificationService(db.GetDB())
	loadService := load.NewLoadService(auditService, notifService)
	chatService := chat.NewChatService(db.GetDB(), notifService, auditService)
	walletService := wallet.NewWalletService(db.GetDB())
	adminService := admin.NewAdminService(db.GetDB())

	// Handlers
	authH := authHandler.NewAuthHandler(authService)
	profileH := profileHandler.NewProfileHandler(profileService)
	productH := productHandler.NewProductHandler(productService)
	orderH := orderHandler.NewOrderHandler(orderService)
	driverH := driverHandler.NewDriverHandler(driverService)
	wsH := wsHandler.NewHandler(notifService, orderService, driverService, db.GetDB(), cfg)
	chatH := chatHandler.NewHandler(chatService)
	walletH := walletHandler.NewWalletHandler(walletService)
	loadH := loadHandler.NewLoadHandler(loadService)
	uploadH := uploadHandler.NewUploadHandler(cfg)
	adminH := adminHandler.NewAdminHandler(adminService, productService)

	uploadDir := "./uploads"
	// Ensure absolute path for logging clarity
	absPath, _ := filepath.Abs(uploadDir)
	log.Printf("Initializing uploads directory at: %s", absPath)

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("CRITICAL: Failed to create uploads directory: %v", err)
	} else {
		// Verify writability
		testFile := filepath.Join(uploadDir, ".write_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			log.Printf("CRITICAL: Uploads directory is NOT writable: %v", err)
		} else {
			os.Remove(testFile)
			log.Printf("Uploads directory is ready and writable.")
		}
	}
	r.Static("/uploads", uploadDir)

	// Public Routes
	r.GET("/", handlers.HealthCheck)
	r.GET("/health", handlers.HealthCheck)

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
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})
		
		// Metrics Endpoint (Optional, guarded by env)
		if cfg.MetricsEnabled {
			api.GET("/metrics", func(c *gin.Context) {
				// Basic runtime stats
				c.JSON(200, gin.H{
					"status": "up",
					"app_env": cfg.AppEnv,
					"metrics": "enabled",
					"websocket": wsH.GetMetrics(),
					// Add real metrics here if prometheus is added
				})
			})
		}
		
		api.GET("/ws", wsH.HandleConnection) // WebSocket Endpoint

		api.GET("/products", productH.ListProducts)
		api.GET("/products/:id", productH.GetProduct)
		api.GET("/vendors", productH.ListVendors)
		api.GET("/vendors/:id", productH.GetVendor)
		api.GET("/delete-account", profileH.DeleteAccountPage)


		authGroup := api.Group("/auth")
		authGroup.Use(middleware.RateLimitMiddleware(rate.Limit(5), 10))
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/admin-login", authH.AdminLogin)
			authGroup.POST("/verify-otp", authH.VerifyOTP)
			authGroup.POST("/forgot-password", authH.ForgotPassword)
			authGroup.POST("/reset-password", authH.ResetPassword)
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
				profileGroup.GET("", profileH.GetProfile)
				profileGroup.GET("/client", profileH.GetProfile)
				profileGroup.GET("/driver", profileH.GetProfile)
				profileGroup.GET("/vendor", profileH.GetProfile)
				
				profileGroup.POST("/client", middleware.RoleMiddleware(string(models.RoleClient)), profileH.CreateClient)
				profileGroup.POST("/driver", middleware.RoleMiddleware(string(models.RoleDriver)), profileH.CreateDriver)
				profileGroup.POST("/vendor", middleware.RoleMiddleware(string(models.RoleVendor)), profileH.CreateVendor)
				profileGroup.PUT("/", profileH.UpdateProfile)
				profileGroup.DELETE("/delete", profileH.DeleteAccount)
			}



			// Client Routes
			clientGroup := protected.Group("/orders")
			clientGroup.Use(middleware.RoleMiddleware(string(models.RoleClient)))
			{
				clientGroup.POST("/checkout", orderH.Checkout)
				clientGroup.PATCH("/:id/cancel", orderH.ClientCancelOrder)
				// Permanently remove a non-active order from the client's history
				clientGroup.DELETE("/:id", orderH.ClientDeleteOrder)
				// "I Have Made Payment" — client self-reports off-platform payment
				clientGroup.POST("/:id/payment-made", orderH.ClientReportPaymentMade)
			}

			// Global Protected Upload Route
			protected.POST("/upload", uploadH.UploadFile)

			// Client Load Routes (admins may also post/manage loads on behalf of customers)
			clientLoadGroup := protected.Group("/loads")
			clientLoadGroup.Use(middleware.RoleMiddleware(string(models.RoleClient), string(models.RoleAdmin)))
			{
				clientLoadGroup.POST("", loadH.CreateLoad)
				clientLoadGroup.GET("/my", loadH.ListMyLoads)
				clientLoadGroup.GET("/:id", loadH.GetLoad)
				clientLoadGroup.POST("/:id/bids/:bid_id/accept", loadH.AcceptLoadBid)
				clientLoadGroup.PATCH("/:id/cancel", loadH.CancelLoad)
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
				jobsGroup.GET("/assigned", driverH.GetAvailableJobs) // Renamed for clarity in this flow
				jobsGroup.POST("/:id/accept", orderH.DriverAcceptLoad)
			}

			// Driver Load Routes
			driverLoadGroup := protected.Group("/loads")
			driverLoadGroup.Use(middleware.RoleMiddleware(string(models.RoleDriver)))
			{
				driverLoadGroup.GET("/available", loadH.ListAvailableLoads)
				driverLoadGroup.GET("/assigned", loadH.GetAssignedLoads)
				driverLoadGroup.POST("/:id/bid", loadH.PlaceLoadBid)
				driverLoadGroup.PATCH("/:id/status", loadH.UpdateLoadStatus)
			}

			// Admin Routes
			adminGroup := protected.Group("/admin")
			adminGroup.Use(middleware.AdminMiddleware())
			{
				adminGroup.PATCH("/drivers/:user_id/approve", profileH.ApproveDriver)
				adminGroup.PATCH("/vendors/:user_id/approve", profileH.ApproveVendor)
				adminGroup.GET("/users", adminH.GetUsers)
				adminGroup.GET("/users/recent", adminH.GetRecentUsers)
				adminGroup.GET("/stats", adminH.GetStats)
				adminGroup.GET("/orders", adminH.GetOrders)
				adminGroup.DELETE("/orders/:id", adminH.DeleteOrder)
				adminGroup.POST("/products", adminH.CreateProduct)
				adminGroup.DELETE("/products/:id", adminH.DeleteProduct)
				adminGroup.DELETE("/users/:id", adminH.DeleteUser)
				adminGroup.POST("/orders/assign-driver", adminH.AssignDriver)
				adminGroup.PATCH("/orders/status", adminH.UpdateOrderStatus)
				// Verify that client's off-platform payment was received → opens order to drivers
				adminGroup.POST("/orders/:id/verify-payment", adminH.VerifyPayment)
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
				vendorGroup.GET("/products", productH.GetMyProducts)
				vendorGroup.PUT("/products/:id", productH.UpdateProduct)
				vendorGroup.DELETE("/products/:id", productH.DeleteProduct)
				vendorGroup.GET("/dashboard", productH.GetVendorDashboard)
				vendorGroup.PATCH("/orders/:id/status", orderH.VendorUpdateStatus)
				vendorGroup.POST("/:id/ready", orderH.VendorMarkReady)
				vendorGroup.POST("/:id/confirm-payment", orderH.ConfirmPayment)
			}

			// Alias for frontend expectation
			protected.GET("/products/vendor/me", productH.GetMyProducts)

			// Wallet Routes
			walletGroup := protected.Group("/wallet")
			{
				walletGroup.GET("/balance", walletH.GetBalance)
				walletGroup.GET("/transactions", walletH.GetTransactions)
			}
		}
	}

	return r
}
