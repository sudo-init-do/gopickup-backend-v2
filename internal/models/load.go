package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoadStatus string

const (
	LoadOpen      LoadStatus = "open"       // Waiting for driver bids
	LoadAssigned  LoadStatus = "assigned"   // Driver accepted
	LoadPickedUp  LoadStatus = "picked_up"  // Driver has the goods
	LoadDelivered LoadStatus = "delivered"  // Completed
	LoadCancelled LoadStatus = "cancelled"  // Cancelled by client
)

type LoadBidStatus string

const (
	LoadBidPending  LoadBidStatus = "pending"
	LoadBidAccepted LoadBidStatus = "accepted"
	LoadBidRejected LoadBidStatus = "rejected"
)

// Load is a direct delivery request posted by a client (no vendor/product needed).
type Load struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ClientID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"client_id"`
	DriverID        *uuid.UUID `gorm:"type:uuid;index" json:"driver_id,omitempty"`
	Title           string     `gorm:"not null" json:"title"`
	Description     string     `gorm:"type:text" json:"description"`
	GoodsType       string     `gorm:"not null" json:"goods_type"`       // e.g. "Cement", "Furniture"
	Weight          *float64   `json:"weight,omitempty"`                  // in kg
	PickupAddress   string     `gorm:"not null" json:"pickup_address"`
	PickupLat       *float64   `json:"pickup_lat,omitempty"`
	PickupLng       *float64   `json:"pickup_lng,omitempty"`
	DeliveryAddress string     `gorm:"not null" json:"delivery_address"`
	DeliveryLat     *float64   `json:"delivery_lat,omitempty"`
	DeliveryLng     *float64   `json:"delivery_lng,omitempty"`
	BudgetAmount    *float64   `gorm:"type:decimal(10,2)" json:"budget_amount,omitempty"` // Client's suggested budget
	AgreedAmount    *float64   `gorm:"type:decimal(10,2)" json:"agreed_amount,omitempty"` // Accepted bid amount
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`                            // Requested delivery date/time (Post Load)
	Status          LoadStatus `gorm:"type:varchar(30);not null;default:'open';index" json:"status"`

	Client   *User      `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	Driver   *User      `gorm:"foreignKey:DriverID" json:"driver,omitempty"`
	Bids     []LoadBid  `gorm:"foreignKey:LoadID" json:"bids,omitempty"`

	CreatedAt time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

func (l *Load) BeforeCreate(tx *gorm.DB) (err error) {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return
}

// LoadBid is a driver's bid on a load.
type LoadBid struct {
	ID       uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	LoadID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"load_id"`
	DriverID uuid.UUID    `gorm:"type:uuid;not null;index" json:"driver_id"`
	Amount   float64      `gorm:"type:decimal(10,2);not null" json:"amount"`
	Note     string       `gorm:"type:text" json:"note,omitempty"`
	Status   LoadBidStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	Driver *User `gorm:"foreignKey:DriverID" json:"driver,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (lb *LoadBid) BeforeCreate(tx *gorm.DB) (err error) {
	if lb.ID == uuid.Nil {
		lb.ID = uuid.New()
	}
	return
}
