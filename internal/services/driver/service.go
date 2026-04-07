package driver

import (
	"errors"
	"fmt"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/audit"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type DriverService struct {
	audit *audit.AuditService
	cfg   *config.Config
}

func NewDriverService(audit *audit.AuditService, cfg *config.Config) *DriverService {
	return &DriverService{audit: audit, cfg: cfg}
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

type AvailableJob struct {
	models.Order
	WhatsAppURL string `json:"whatsapp_url"` // Opens GoPickup support to negotiate delivery fee
}

// GetAvailableJobs returns all orders in `processing` status (payment verified, awaiting driver)
// to any approved driver. The driver negotiates delivery fee on WhatsApp then taps Accept.
func (s *DriverService) GetAvailableJobs(driverID uuid.UUID) ([]AvailableJob, error) {
	var profile models.DriverProfile
	if err := db.GetDB().First(&profile, "user_id = ?", driverID).Error; err != nil {
		return []AvailableJob{}, nil
	}
	if !profile.IsApproved {
		return []AvailableJob{}, nil
	}

	var orders []models.Order
	// Show orders that are:
	// a) open to all approved drivers (processing), OR
	// b) specifically pre-assigned to this driver by admin (assigned)
	err := db.GetDB().
		Preload("Vendor").Preload("Items").
		Where("status = ? OR (status = ? AND driver_id = ?)",
			models.OrderProcessing, models.OrderAssigned, driverID).
		Order("created_at DESC").
		Find(&orders).Error
	if err != nil {
		return nil, err
	}

	// Build support WhatsApp URL for each job
	supportPhone := strings.TrimPrefix(s.cfg.WhatsAppSupportNumber, "+")
	if supportPhone == "" {
		supportPhone = "2348000000000"
	}

	jobs := make([]AvailableJob, 0, len(orders))
	for _, o := range orders {
		msg := fmt.Sprintf(
			"Hi GoPickup Support! I'm interested in a delivery job.\n\nOrder ID: %s\nPickup: %s\nDelivery: %s\n\nPlease share the delivery fee details.",
			o.ID.String()[:8], o.PickupAddress, o.DeliveryAddress,
		)
		wwURL := fmt.Sprintf("https://wa.me/%s?text=%s", supportPhone, url.QueryEscape(msg))
		jobs = append(jobs, AvailableJob{Order: o, WhatsAppURL: wwURL})
	}
	return jobs, nil
}

func (s *DriverService) PlaceBid(driverID uuid.UUID, orderID uuid.UUID, amount float64) (*models.Bid, error) {
	return nil, errors.New("bidding is disabled. negotiate via WhatsApp then tap Accept")
}
