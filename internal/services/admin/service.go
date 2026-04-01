package admin

import (
	"errors"
	"gopickup/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

type PlatformStats struct {
	TotalUsers     int64 `json:"total_users"`
	TotalClients   int64 `json:"total_clients"`
	TotalDrivers   int64 `json:"total_drivers"`
	TotalVendors   int64 `json:"total_vendors"`
	TotalOrders    int64 `json:"total_orders"`
	PendingOrders  int64 `json:"pending_orders"`
	ActiveOrders   int64 `json:"active_orders"`
	TotalProducts  int64 `json:"total_products"`
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
	if err := s.db.Order("created_at desc").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
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
