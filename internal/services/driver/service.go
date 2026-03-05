package driver

import (
	"errors"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/notification"

	"github.com/google/uuid"
)

type DriverService struct{}

func NewDriverService() *DriverService {
	return &DriverService{}
}

func (s *DriverService) UpdateLocation(driverID uuid.UUID, lat, lng float64) error {
	result := db.GetDB().Model(&models.DriverProfile{}).
		Where("user_id = ?", driverID).
		Updates(map[string]interface{}{
			"current_location_lat": lat,
			"current_location_lng": lng,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("driver profile not found")
	}
	return nil
}

func (s *DriverService) GetAvailableJobs(driverID uuid.UUID) ([]models.Order, error) {
	var profile models.DriverProfile
	if err := db.GetDB().First(&profile, "user_id = ?", driverID).Error; err != nil {
		return nil, errors.New("driver profile not found")
	}
	if !profile.IsApproved {
		return nil, errors.New("driver not approved")
	}

	var orders []models.Order
	// Return orders waiting for driver
	err := db.GetDB().Where("status = ?", models.OrderSearchingDriver).Find(&orders).Error
	return orders, err
}

func (s *DriverService) PlaceBid(driverID uuid.UUID, orderID uuid.UUID, amount float64) (*models.Bid, error) {
	var profile models.DriverProfile
	if err := db.GetDB().First(&profile, "user_id = ?", driverID).Error; err != nil {
		return nil, errors.New("driver profile not found")
	}
	if !profile.IsApproved {
		return nil, errors.New("driver not approved")
	}

	var order models.Order
	if err := db.GetDB().First(&order, "id = ?", orderID).Error; err != nil {
		return nil, errors.New("order not found")
	}
	if order.Status != models.OrderSearchingDriver {
		return nil, errors.New("order not available for bidding")
	}

	// Check if bid exists
	var existingBid models.Bid
	err := db.GetDB().Where("order_id = ? AND driver_id = ?", orderID, driverID).First(&existingBid).Error
	if err == nil {
		// Update existing bid
		existingBid.Amount = amount
		if err := db.GetDB().Save(&existingBid).Error; err != nil {
			return nil, err
		}
		// Notify client about updated bid
		notification.GetService().NotifyNewBid(order.ClientID, order.ID, existingBid.Amount)
		return &existingBid, nil
	}

	bid := &models.Bid{
		OrderID:  orderID,
		DriverID: driverID,
		Amount:   amount,
		Status:   models.BidPending,
	}
	if err := db.GetDB().Create(bid).Error; err != nil {
		return nil, err
	}
	// Notify client about new bid
	notification.GetService().NotifyNewBid(order.ClientID, order.ID, bid.Amount)
	return bid, nil
}
