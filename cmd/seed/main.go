package main

import (
	"flag"
	"fmt"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. Parse flags
	seedAll := flag.Bool("seed", false, "Seed all test accounts (Admin, Client, Driver, Vendor)")
	createAdmin := flag.Bool("admin", false, "Create an admin user only")
	verifyUser := flag.String("verify", "", "Email of user to verify (mark verified=true)")
	emailFlag := flag.String("email", "admin@gopickup.com", "Email for admin user")
	passwordFlag := flag.String("password", "AdminPassword123!", "Password for admin user")
	flag.Parse()

	// 2. Load Config & Connect DB
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Safety check
	allowSeed := os.Getenv("ALLOW_SEED")
	if cfg.AppEnv == "production" && allowSeed != "true" {
		log.Fatal("ERROR: Seeding is disabled in PRODUCTION. Set ALLOW_SEED=true to force.")
	}

	db.Connect(cfg)

	// 3. Execute requested action
	if *seedAll {
		seedAllAccounts()
	} else if *createAdmin {
		createAdminUser(*emailFlag, *passwordFlag)
	} else if *verifyUser != "" {
		verifyUserByEmail(*verifyUser)
	} else {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/seed/main.go -seed (Create all test accounts)")
		fmt.Println("  go run cmd/seed/main.go -admin -email=... -password=...")
		fmt.Println("  go run cmd/seed/main.go -verify=user@example.com")
		os.Exit(1)
	}
}

func seedAllAccounts() {
	log.Println("Seeding test accounts...")

	// 1. Admin
	createAdminUser("admin@test.com", "Password123!")

	// 2. Vendor
	vendorID := createVendorUser("vendor@test.com", "Password123!", "Tech Haven", "08055555555", "Electronics & Gadgets")
	var products []models.Product
	if vendorID != uuid.Nil {
		products = seedProducts(vendorID)
	}

	// 3. Drivers
	// Driver 1: Approved and Verified
	driver1ID := createDriverUser("driver@test.com", "Password123!", "John Doe", "08098765432", "LAG-123-XY", true)
	// Driver 2: Unapproved (for testing approval)
	createDriverUser("newdriver@test.com", "Password123!", "Jane Smith", "08098765433", "ABJ-456-YZ", false)

	// 4. Client
	clientID := createClientUser("client@test.com", "Password123!", "Alice Wonderland", "08012345678")

	// 5. Orders & Interactions
	if clientID != uuid.Nil && vendorID != uuid.Nil && len(products) > 0 {
		seedOrders(clientID, vendorID, driver1ID, products)
		seedWallets(clientID, driver1ID)
		seedChats(clientID, vendorID, driver1ID)
	}

	log.Println("Seeding complete! Log in with:")
	log.Println("  Client: client@test.com / Password123!")
	log.Println("  Driver: driver@test.com / Password123!")
	log.Println("  Vendor: vendor@test.com / Password123!")
	log.Println("  Admin:  admin@test.com  / Password123!")
}

func seedWallets(clientID, driverID uuid.UUID) {
	log.Println("Seeding wallets...")
	
	// Client Wallet
	clientWallet := models.Wallet{
		UserID:   clientID,
		Balance:  50000.00,
		Currency: "NGN",
	}
	db.GetDB().Where(models.Wallet{UserID: clientID}).FirstOrCreate(&clientWallet)
	
	// Client Transactions
	db.GetDB().Create(&models.Transaction{
		WalletID:    clientWallet.ID,
		Amount:      50000.00,
		Type:        models.TransactionCredit,
		Description: "Wallet Funding",
		Status:      "success",
		Reference:   "REF-" + uuid.New().String()[:8],
	})
	db.GetDB().Create(&models.Transaction{
		WalletID:    clientWallet.ID,
		Amount:      2500.00,
		Type:        models.TransactionDebit,
		Description: "Payment for Order #1234",
		Status:      "success",
		Reference:   "REF-" + uuid.New().String()[:8],
	})

	// Driver Wallet
	driverWallet := models.Wallet{
		UserID:   driverID,
		Balance:  12500.00,
		Currency: "NGN",
	}
	db.GetDB().Where(models.Wallet{UserID: driverID}).FirstOrCreate(&driverWallet)

	// Driver Transactions
	db.GetDB().Create(&models.Transaction{
		WalletID:    driverWallet.ID,
		Amount:      4500.00,
		Type:        models.TransactionCredit,
		Description: "Earnings for Order #5678",
		Status:      "success",
		Reference:   "REF-" + uuid.New().String()[:8],
	})
}

func seedChats(clientID, vendorID, driverID uuid.UUID) {
	log.Println("Seeding chats...")

	// Chat 1: Client <-> Driver
	chat1 := models.Chat{
		Participants: []models.User{{ID: clientID}, {ID: driverID}},
	}
	if err := db.GetDB().Create(&chat1).Error; err == nil {
		// Add Messages
		messages := []models.Message{
			{ChatID: chat1.ID, SenderID: clientID, Content: "Hi, where are you now?", IsRead: true, CreatedAt: time.Now().Add(-10 * time.Minute)},
			{ChatID: chat1.ID, SenderID: driverID, Content: "I'm 5 mins away from pickup.", IsRead: true, CreatedAt: time.Now().Add(-9 * time.Minute)},
			{ChatID: chat1.ID, SenderID: clientID, Content: "Okay, thanks!", IsRead: false, CreatedAt: time.Now().Add(-5 * time.Minute)},
		}
		for _, m := range messages {
			db.GetDB().Create(&m)
		}
		
		// Link participants manually since GORM might not handle the many2many creation perfectly with just ID structs in all versions
		db.GetDB().Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING", chat1.ID, clientID)
		db.GetDB().Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING", chat1.ID, driverID)
	}

	// Chat 2: Client <-> Vendor
	chat2 := models.Chat{
		Participants: []models.User{{ID: clientID}, {ID: vendorID}},
	}
	if err := db.GetDB().Create(&chat2).Error; err == nil {
		db.GetDB().Create(&models.Message{
			ChatID: chat2.ID, SenderID: clientID, Content: "Is the black headphone in stock?", IsRead: true, CreatedAt: time.Now().Add(-1 * time.Hour),
		})
		db.GetDB().Create(&models.Message{
			ChatID: chat2.ID, SenderID: vendorID, Content: "Yes, we have 5 left.", IsRead: false, CreatedAt: time.Now().Add(-50 * time.Minute),
		})
		
		db.GetDB().Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING", chat2.ID, clientID)
		db.GetDB().Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING", chat2.ID, vendorID)
	}
}

func seedOrders(clientID, vendorID, driverID uuid.UUID, products []models.Product) {
	if len(products) < 3 {
		return
	}

	orders := []models.Order{
		// Order 1: Pending
		{
			ClientID:           clientID,
			VendorID:           vendorID,
			TotalProductAmount: products[0].Price,
			PaymentMethod:      models.PaymentCard,
			PickupAddress:      "456 Market St, Lagos",
			DeliveryAddress:    "123 Client St, Lagos",
			Status:             models.OrderPending,
			Items: []models.OrderItem{
				{ProductID: products[0].ID, Name: products[0].Name, Price: products[0].Price, Quantity: 1},
			},
		},
		// Order 2: Processing
		{
			ClientID:           clientID,
			VendorID:           vendorID,
			TotalProductAmount: products[1].Price * 2,
			PaymentMethod:      models.PaymentWallet,
			PickupAddress:      "456 Market St, Lagos",
			DeliveryAddress:    "789 Another St, Abuja",
			Status:             models.OrderProcessing,
			Items: []models.OrderItem{
				{ProductID: products[1].ID, Name: products[1].Name, Price: products[1].Price, Quantity: 2},
			},
		},
		// Order 3: Searching Driver (Ready for Driver Dashboard)
		{
			ClientID:           clientID,
			VendorID:           vendorID,
			TotalProductAmount: products[2].Price,
			PaymentMethod:      models.PaymentCard,
			PickupAddress:      "Shop 12, Computer Village, Ikeja",
			DeliveryAddress:    "15 Admiralty Way, Lekki",
			Status:             models.OrderSearchingDriver,
			Items: []models.OrderItem{
				{ProductID: products[2].ID, Name: products[2].Name, Price: products[2].Price, Quantity: 1},
			},
		},
		// Order 4: Delivered (History)
		{
			ClientID:           clientID,
			VendorID:           vendorID,
			DriverID:           &driverID,
			TotalProductAmount: products[0].Price + products[1].Price,
			PaymentMethod:      models.PaymentWallet,
			PickupAddress:      "456 Market St, Lagos",
			DeliveryAddress:    "123 Client St, Lagos",
			Status:             models.OrderDelivered,
			Items: []models.OrderItem{
				{ProductID: products[0].ID, Name: products[0].Name, Price: products[0].Price, Quantity: 1},
				{ProductID: products[1].ID, Name: products[1].Name, Price: products[1].Price, Quantity: 1},
			},
			CreatedAt: time.Now().Add(-24 * time.Hour), // Yesterday
		},
	}

	for _, o := range orders {
		o.ID = uuid.New()
		for i := range o.Items {
			o.Items[i].ID = uuid.New()
			o.Items[i].OrderID = o.ID
		}
		
		if err := db.GetDB().Create(&o).Error; err != nil {
			log.Printf("Failed to create order: %v", err)
		} else {
			log.Printf("Order created: %s (Status: %s)", o.ID, o.Status)
		}
	}
}

func seedProducts(vendorID uuid.UUID) []models.Product {
	products := []models.Product{
		{
			VendorID:      vendorID,
			Name:          "Premium Headphones",
			Description:   "Noise cancelling headphones with superior sound quality",
			Price:         25000.00,
			Category:      "Electronics",
			StockQuantity: 50,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=500&auto=format&fit=crop&q=60",
		},
		{
			VendorID:      vendorID,
			Name:          "Organic Coffee Beans",
			Description:   "Freshly roasted organic coffee beans, 1kg pack",
			Price:         5000.00,
			Category:      "Food",
			StockQuantity: 100,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1559056199-641a0ac8b55e?w=500&auto=format&fit=crop&q=60",
		},
		{
			VendorID:      vendorID,
			Name:          "Running Shoes",
			Description:   "Comfortable running shoes for all terrains",
			Price:         15000.00,
			Category:      "Fashion",
			StockQuantity: 25,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=500&auto=format&fit=crop&q=60",
		},
		{
			VendorID:      vendorID,
			Name:          "Smart Watch",
			Description:   "Track your fitness and notifications on the go",
			Price:         45000.00,
			Category:      "Electronics",
			StockQuantity: 30,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=500&auto=format&fit=crop&q=60",
		},
		{
			VendorID:      vendorID,
			Name:          "Leather Backpack",
			Description:   "Durable and stylish leather backpack for work or travel",
			Price:         35000.00,
			Category:      "Fashion",
			StockQuantity: 15,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=500&auto=format&fit=crop&q=60",
		},
		{
			VendorID:      vendorID,
			Name:          "Dangote Cement",
			Description:   "High-quality cement for all construction needs",
			Price:         8500.00,
			Category:      "Cement",
			StockQuantity: 500,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=500&auto=format&fit=crop&q=60",
		},
		{
			VendorID:      vendorID,
			Name:          "Reinforcement Steel Bars",
			Description:   "12mm TMT steel bars for structural strength",
			Price:         12000.00,
			Category:      "Steel",
			StockQuantity: 200,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1533000932845-97191147076d?w=500&auto=format&fit=crop&q=60",
		},
		{
			VendorID:      vendorID,
			Name:          "Plumbing Pipes (PVC)",
			Description:   "Durable PVC pipes for drainage and water supply",
			Price:         2500.00,
			Category:      "Plumbing",
			StockQuantity: 150,
			IsActive:      true,
			ImageURL:      "https://images.unsplash.com/photo-1581094288338-2314dddb7ec4?w=500&auto=format&fit=crop&q=60",
		},
	}

	for i := range products {
		products[i].ID = uuid.New()
		if err := db.GetDB().Create(&products[i]).Error; err != nil {
			log.Printf("Failed to create product %s: %v", products[i].Name, err)
		} else {
			log.Printf("Product created: %s", products[i].Name)
		}
	}
	return products
}

func createAdminUser(email, password string) {
	createUser(email, password, "admin", true)
	log.Printf("Admin user created/verified: %s", email)
}

func createClientUser(email, password, name, phone string) uuid.UUID {
	userID := createUser(email, password, "client", true)
	if userID != uuid.Nil {
		profile := models.ClientProfile{
			UserID:      userID,
			FullName:    name,
			PhoneNumber: phone,
			Address:     "123 Client St, Lagos",
		}
		db.GetDB().FirstOrCreate(&profile, models.ClientProfile{UserID: userID})
		log.Printf("Client user created: %s", email)
	}
	return userID
}

func createDriverUser(email, password, name, phone, plate string, approved bool) uuid.UUID {
	userID := createUser(email, password, "driver", true)
	if userID != uuid.Nil {
		profile := models.DriverProfile{
			UserID:          userID,
			FullName:        name,
			PhoneNumber:     phone,
			LicenseNumber:   "LIC-" + plate,
			VehicleType:     models.VehicleVan,
			PlateNumber:     plate,
			VehicleCapacity: 1000,
			IsApproved:      approved,
		}
		db.GetDB().FirstOrCreate(&profile, models.DriverProfile{UserID: userID})
		log.Printf("Driver user created: %s (Approved: %v)", email, approved)
	}
	return userID
}

func createVendorUser(email, password, storeName, phone, businessType string) uuid.UUID {
	userID := createUser(email, password, "vendor", true)
	if userID != uuid.Nil {
		profile := models.VendorProfile{
			UserID:       userID,
			StoreName:    storeName,
			PhoneNumber:  phone,
			BusinessType: businessType,
			Address:      "456 Market St, Lagos",
			IsApproved:   true,
		}
		db.GetDB().FirstOrCreate(&profile, models.VendorProfile{UserID: userID})
		log.Printf("Vendor user created: %s", email)
	}
	return userID
}

func createUser(email, password, role string, isVerified bool) uuid.UUID {
	// Check if exists
	var existing models.User
	if err := db.GetDB().Where("email = ?", email).First(&existing).Error; err == nil {
		return existing.ID
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	userID := uuid.New()
	user := models.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         models.UserRole(role),
		IsVerified:   isVerified,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.GetDB().Create(&user).Error; err != nil {
		log.Printf("Failed to create user %s: %v", email, err)
		return uuid.Nil
	}
	return userID
}

func verifyUserByEmail(email string) {
	if err := db.GetDB().Model(&models.User{}).Where("email = ?", email).Update("is_verified", true).Error; err != nil {
		log.Printf("Failed to verify user %s: %v", email, err)
	} else {
		log.Printf("User %s verified successfully", email)
	}
}
