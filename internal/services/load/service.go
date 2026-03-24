package load

import (
	"errors"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/audit"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoadService struct {
	audit *audit.AuditService
}

func NewLoadService(audit *audit.AuditService) *LoadService {
	return &LoadService{audit: audit}
}

// --- DTOs ---

type CreateLoadRequest struct {
	Title           string   `json:"title" binding:"required"`
	Description     string   `json:"description"`
	GoodsType       string   `json:"goods_type" binding:"required"`
	Weight          *float64 `json:"weight"`
	PickupAddress   string   `json:"pickup_address" binding:"required"`
	PickupLat       *float64 `json:"pickup_lat"`
	PickupLng       *float64 `json:"pickup_lng"`
	DeliveryAddress string   `json:"delivery_address" binding:"required"`
	DeliveryLat     *float64 `json:"delivery_lat"`
	DeliveryLng     *float64 `json:"delivery_lng"`
	BudgetAmount    *float64 `json:"budget_amount"`
}

type PlaceLoadBidRequest struct {
	Amount float64 `json:"amount" binding:"required,min=1"`
	Note   string  `json:"note"`
}

// --- Client Methods ---

// CreateLoad allows a client to post a new load request.
func (s *LoadService) CreateLoad(clientID uuid.UUID, req CreateLoadRequest) (*models.Load, error) {
	load := models.Load{
		ClientID:        clientID,
		Title:           req.Title,
		Description:     req.Description,
		GoodsType:       req.GoodsType,
		Weight:          req.Weight,
		PickupAddress:   req.PickupAddress,
		PickupLat:       req.PickupLat,
		PickupLng:       req.PickupLng,
		DeliveryAddress: req.DeliveryAddress,
		DeliveryLat:     req.DeliveryLat,
		DeliveryLng:     req.DeliveryLng,
		BudgetAmount:    req.BudgetAmount,
		Status:          models.LoadOpen,
	}

	if err := db.GetDB().Create(&load).Error; err != nil {
		return nil, err
	}

	s.audit.Log(clientID, "LOAD_CREATED", "load", load.ID, nil)
	return &load, nil
}

// ListClientLoads returns all loads posted by this client.
func (s *LoadService) ListClientLoads(clientID uuid.UUID, page, limit int) ([]models.Load, int64, error) {
	if page <= 0 { page = 1 }
	if limit <= 0 { limit = 10 }
	if limit > 100 { limit = 100 }
	offset := (page - 1) * limit

	var loads []models.Load
	var count int64
	q := db.GetDB().Model(&models.Load{}).Where("client_id = ?", clientID).Order("created_at DESC")
	q.Count(&count)
	if err := q.Preload("Bids.Driver.DriverProfile").Offset(offset).Limit(limit).Find(&loads).Error; err != nil {
		return nil, 0, err
	}
	return loads, count, nil
}

// GetLoad returns a specific load if the requester is the client owner.
func (s *LoadService) GetLoad(clientID uuid.UUID, loadID uuid.UUID) (*models.Load, error) {
	var load models.Load
	if err := db.GetDB().Preload("Bids.Driver.DriverProfile").First(&load, "id = ?", loadID).Error; err != nil {
		return nil, errors.New("load not found")
	}
	if load.ClientID != clientID {
		return nil, errors.New("forbidden")
	}
	return &load, nil
}

// AcceptLoadBid - client accepts a driver's bid on their load.
func (s *LoadService) AcceptLoadBid(clientID uuid.UUID, loadID uuid.UUID, bidID uuid.UUID) (*models.Load, error) {
	var load models.Load

	err := db.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&load, "id = ?", loadID).Error; err != nil {
			return errors.New("load not found")
		}
		if load.ClientID != clientID {
			return errors.New("forbidden")
		}
		if load.Status != models.LoadOpen {
			return errors.New("load is no longer open for bids")
		}

		var bid models.LoadBid
		if err := tx.First(&bid, "id = ?", bidID).Error; err != nil {
			return errors.New("bid not found")
		}
		if bid.LoadID != loadID {
			return errors.New("bid does not belong to this load")
		}

		// Assign driver, set agreed amount, update status
		load.DriverID = &bid.DriverID
		load.AgreedAmount = &bid.Amount
		load.Status = models.LoadAssigned
		if err := tx.Save(&load).Error; err != nil {
			return err
		}

		// Accept this bid
		if err := tx.Model(&bid).Update("status", models.LoadBidAccepted).Error; err != nil {
			return err
		}

		// Reject all other bids
		return tx.Model(&models.LoadBid{}).
			Where("load_id = ? AND id != ?", loadID, bidID).
			Update("status", models.LoadBidRejected).Error
	})

	if err != nil {
		return nil, err
	}
	s.audit.Log(clientID, "LOAD_BID_ACCEPTED", "load", loadID, nil)
	return &load, nil
}

// CancelLoad - client cancels an open load.
func (s *LoadService) CancelLoad(clientID uuid.UUID, loadID uuid.UUID) (*models.Load, error) {
	var load models.Load
	if err := db.GetDB().First(&load, "id = ?", loadID).Error; err != nil {
		return nil, errors.New("load not found")
	}
	if load.ClientID != clientID {
		return nil, errors.New("forbidden")
	}
	if load.Status != models.LoadOpen {
		return nil, errors.New("only open loads can be cancelled")
	}
	load.Status = models.LoadCancelled
	if err := db.GetDB().Save(&load).Error; err != nil {
		return nil, err
	}
	s.audit.Log(clientID, "LOAD_CANCELLED", "load", loadID, nil)
	return &load, nil
}

// --- Driver Methods ---

// ListAvailableLoads returns all open loads for drivers.
func (s *LoadService) ListAvailableLoads(page, limit int) ([]models.Load, int64, error) {
	if page <= 0 { page = 1 }
	if limit <= 0 { limit = 10 }
	if limit > 100 { limit = 100 }
	offset := (page - 1) * limit

	var loads []models.Load
	var count int64
	q := db.GetDB().Model(&models.Load{}).Where("status = ?", models.LoadOpen).Order("created_at DESC")
	q.Count(&count)
	if err := q.Offset(offset).Limit(limit).Find(&loads).Error; err != nil {
		return nil, 0, err
	}
	return loads, count, nil
}

// PlaceLoadBid - driver places a bid on an open load.
func (s *LoadService) PlaceLoadBid(driverID uuid.UUID, loadID uuid.UUID, req PlaceLoadBidRequest) (*models.LoadBid, error) {
	var load models.Load
	if err := db.GetDB().First(&load, "id = ?", loadID).Error; err != nil {
		return nil, errors.New("load not found")
	}
	if load.Status != models.LoadOpen {
		return nil, errors.New("load is not open for bidding")
	}

	// Prevent duplicate bids from same driver
	var existing models.LoadBid
	if err := db.GetDB().Where("load_id = ? AND driver_id = ? AND status = ?", loadID, driverID, models.LoadBidPending).First(&existing).Error; err == nil {
		return nil, errors.New("you have already placed a bid on this load")
	}

	bid := models.LoadBid{
		LoadID:   loadID,
		DriverID: driverID,
		Amount:   req.Amount,
		Note:     req.Note,
		Status:   models.LoadBidPending,
	}

	if err := db.GetDB().Create(&bid).Error; err != nil {
		return nil, err
	}

	s.audit.Log(driverID, "LOAD_BID_PLACED", "load", loadID, nil)
	return &bid, nil
}

// UpdateLoadStatus - driver marks load as picked_up or delivered.
func (s *LoadService) UpdateLoadStatus(driverID uuid.UUID, loadID uuid.UUID, status models.LoadStatus) (*models.Load, error) {
	var load models.Load
	if err := db.GetDB().First(&load, "id = ?", loadID).Error; err != nil {
		return nil, errors.New("load not found")
	}
	if load.DriverID == nil || *load.DriverID != driverID {
		return nil, errors.New("forbidden")
	}

	// Enforce valid transitions
	switch load.Status {
	case models.LoadAssigned:
		if status != models.LoadPickedUp {
			return nil, errors.New("invalid transition: can only move to picked_up")
		}
	case models.LoadPickedUp:
		if status != models.LoadDelivered {
			return nil, errors.New("invalid transition: can only move to delivered")
		}
	default:
		return nil, errors.New("invalid transition")
	}

	load.Status = status
	if err := db.GetDB().Save(&load).Error; err != nil {
		return nil, err
	}
	s.audit.Log(driverID, "LOAD_STATUS_UPDATED", "load", loadID, map[string]interface{}{"status": status})
	return &load, nil
}
