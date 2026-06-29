package admin

import (
	"errors"
	"gopickup/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(database *gorm.DB) *AdminService {
	return &AdminService{
		db: database,
	}
}

func (s *AdminService) GetUsers(role string) ([]models.User, error) {
	var users []models.User
	query := s.db.Preload("ClientProfile").Preload("DriverProfile").Preload("VendorProfile")

	if role != "" {
		query = query.Where("role = ?", role)
	}

	if err := query.Order("created_at desc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AdminService) GetRecentUsers(limit int) ([]models.User, error) {
	var users []models.User
	if limit <= 0 {
		limit = 10
	}
	if err := s.db.Order("created_at desc").Limit(limit).
		Preload("ClientProfile").
		Preload("DriverProfile").
		Preload("VendorProfile").
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

type PlatformStats struct {
	TotalUsers    int64 `json:"total_users"`
	TotalClients  int64 `json:"total_clients"`
	TotalDrivers  int64 `json:"total_drivers"`
	TotalVendors  int64 `json:"total_vendors"`
	TotalOrders   int64 `json:"total_orders"`
	PendingOrders int64 `json:"pending_orders"`
	ActiveOrders  int64 `json:"active_orders"`
	TotalProducts int64 `json:"total_products"`
}

func (s *AdminService) GetStats() (*PlatformStats, error) {
	var stats PlatformStats

	s.db.Model(&models.User{}).Count(&stats.TotalUsers)
	s.db.Model(&models.User{}).Where("role = ?", models.RoleClient).Count(&stats.TotalClients)
	s.db.Model(&models.User{}).Where("role = ?", models.RoleDriver).Count(&stats.TotalDrivers)
	s.db.Model(&models.User{}).Where("role = ?", models.RoleVendor).Count(&stats.TotalVendors)

	s.db.Model(&models.Order{}).Count(&stats.TotalOrders)
	s.db.Model(&models.Order{}).Where("status = ?", models.OrderPending).Count(&stats.PendingOrders)
	s.db.Model(&models.Order{}).Where("status IN (?)", []models.OrderStatus{
		models.OrderProcessing,
		models.OrderAssigned,
		models.OrderInProgress,
		models.OrderPickedUp,
		models.OrderOnTheWay,
	}).Count(&stats.ActiveOrders)

	s.db.Model(&models.Product{}).Count(&stats.TotalProducts)

	return &stats, nil
}

func (s *AdminService) GetOrders() ([]models.Order, error) {
	var orders []models.Order
	if err := s.db.
		Order("created_at desc").
		Preload("Client").
		Preload("Client.ClientProfile").
		Preload("Vendor").
		Preload("Items").
		Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// GetLoads returns every load (Book Driver + Post Load requests) for the admin
// console, newest first, with client/driver/bid details preloaded so the team
// can call drivers to come bid.
func (s *AdminService) GetLoads() ([]models.Load, error) {
	var loads []models.Load
	if err := s.db.
		Order("created_at desc").
		Preload("Client").
		Preload("Client.ClientProfile").
		Preload("Driver").
		Preload("Driver.DriverProfile").
		Preload("Bids").
		Preload("Bids.Driver.DriverProfile").
		Find(&loads).Error; err != nil {
		return nil, err
	}
	return loads, nil
}

func (s *AdminService) AssignDriver(orderID uuid.UUID, driverID uuid.UUID, agreedPrice float64, deliveryFee float64) error {
	var driver models.User
	if err := s.db.Where("id = ? AND role = ?", driverID, models.RoleDriver).First(&driver).Error; err != nil {
		return errors.New("driver not found")
	}

	return s.db.Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"driver_id":           driverID,
		"status":              models.OrderAssigned,
		"agreed_price":        agreedPrice,
		"agreed_delivery_fee": deliveryFee,
	}).Error
}

func (s *AdminService) UpdateOrderStatus(orderID uuid.UUID, status models.OrderStatus) error {
	return s.db.Model(&models.Order{}).Where("id = ?", orderID).Update("status", status).Error
}

// VerifyPayment is called by admin after confirming the client's payment off-platform.
// Moves order from payment_made → processing, making it visible to all approved drivers.
func (s *AdminService) VerifyPayment(orderID uuid.UUID, agreedPrice *float64) error {
	var o models.Order
	if err := s.db.First(&o, "id = ?", orderID).Error; err != nil {
		return errors.New("order not found")
	}
	if o.Status != models.OrderPaymentMade {
		return errors.New("order is not awaiting payment verification")
	}
	updates := map[string]interface{}{"status": models.OrderProcessing}
	if agreedPrice != nil {
		updates["agreed_price"] = *agreedPrice
	}
	return s.db.Model(&models.Order{}).Where("id = ?", orderID).Updates(updates).Error
}

func (s *AdminService) DeleteUser(userID uuid.UUID) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return errors.New("user not found")
	}

	// Hard delete related data manually to ensure complete cleanup
	s.db.Unscoped().Where("user_id = ?", userID).Delete(&models.ClientProfile{})
	s.db.Unscoped().Where("user_id = ?", userID).Delete(&models.DriverProfile{})
	s.db.Unscoped().Where("user_id = ?", userID).Delete(&models.VendorProfile{})
	s.db.Unscoped().Where("vendor_id = ?", userID).Delete(&models.Product{})

	// Hard delete the user
	if err := s.db.Unscoped().Where("id = ?", userID).Delete(&models.User{}).Error; err != nil {
		return err
	}

	return nil
}

// DeleteOrder permanently removes an order regardless of status. Admin-only
// operation; reserved stock is restored for pre-fulfillment statuses. Order
// items and bids are cascaded.
func (s *AdminService) DeleteOrder(orderID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var o models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, "id = ?", orderID).Error; err != nil {
			return err
		}
		switch o.Status {
		case models.OrderPending, models.OrderAwaitingPayment, models.OrderPaymentMade, models.OrderProcessing:
			if err := s.restoreStockForOrderTx(tx, o.ID); err != nil {
				return err
			}
		}
		if err := tx.Where("order_id = ?", orderID).Delete(&models.Bid{}).Error; err != nil {
			return err
		}
		if err := tx.Where("order_id = ?", orderID).Delete(&models.OrderItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Order{}, "id = ?", orderID).Error
	})
}

// restoreStockForOrderTx adds order item quantities back to product stock.
func (s *AdminService) restoreStockForOrderTx(tx *gorm.DB, orderID uuid.UUID) error {
	var items []models.OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&items).Error; err != nil {
		return err
	}
	for _, it := range items {
		if err := tx.Model(&models.Product{}).
			Where("id = ?", it.ProductID).
			Update("stock_quantity", gorm.Expr("stock_quantity + ?", it.Quantity)).
			Error; err != nil {
			return err
		}
	}
	return nil
}
