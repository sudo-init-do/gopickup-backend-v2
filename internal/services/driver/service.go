package driver

import (
	"errors"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/audit"
	"github.com/google/uuid"
)

type DriverService struct {
	audit *audit.AuditService
}

func NewDriverService(audit *audit.AuditService) *DriverService {
	return &DriverService{audit: audit}
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
		// Return empty list instead of error to prevent frontend crash if profile missing
		return []models.Order{}, nil
	}
	if !profile.IsApproved {
		return []models.Order{}, nil
	}

	var orders []models.Order
	// Return orders assigned to THIS specific driver waiting for acceptance
	err := db.GetDB().Where("driver_id = ? AND status = ?", driverID, models.OrderAssigned).Find(&orders).Error
	return orders, err
}

func (s *DriverService) PlaceBid(driverID uuid.UUID, orderID uuid.UUID, amount float64) (*models.Bid, error) {
	return nil, errors.New("bidding is disabled. loads are now assigned directly by admin")
}
