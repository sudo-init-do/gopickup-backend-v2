package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderPending         OrderStatus = "pending"
	OrderProcessing      OrderStatus = "processing"
	OrderSearchingDriver OrderStatus = "searching_driver"
	OrderAssigned        OrderStatus = "assigned"
	OrderPickedUp        OrderStatus = "picked_up"
	OrderDelivered       OrderStatus = "delivered"
	OrderCancelled       OrderStatus = "cancelled"
)

type BidStatus string

const (
	BidPending  BidStatus = "pending"
	BidAccepted BidStatus = "accepted"
	BidRejected BidStatus = "rejected"
)

type Bid struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	DriverID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Amount    float64   `gorm:"type:decimal(10,2);not null"`
	Status    BidStatus `gorm:"type:varchar(20);default:'pending'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (b *Bid) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return
}

type PaymentMethod string

const (
	PaymentWallet         PaymentMethod = "wallet"
	PaymentCard           PaymentMethod = "card"
	PaymentCashOnDelivery PaymentMethod = "cash_on_delivery"
)

type Order struct {
	ID                 uuid.UUID     `gorm:"type:uuid;primaryKey"`
	ClientID           uuid.UUID     `gorm:"type:uuid;not null;index"`
	VendorID           uuid.UUID     `gorm:"type:uuid;not null;index"`
	DriverID           *uuid.UUID    `gorm:"type:uuid;index"`
	TotalProductAmount float64       `gorm:"type:decimal(10,2);not null"`
	PaymentMethod      PaymentMethod `gorm:"type:varchar(30);not null"`
	PickupAddress      string        `gorm:"not null"`
	DeliveryAddress    string        `gorm:"not null"`
	DeliveryLat        *float64
	DeliveryLng        *float64
	Status             OrderStatus `gorm:"type:varchar(30);not null;index"`

	Items []OrderItem `gorm:"foreignKey:OrderID"`
	Bids  []Bid       `gorm:"foreignKey:OrderID"`

	CreatedAt          time.Time   `gorm:"index"` // Indexed for sorting
	UpdatedAt          time.Time
	DeletedAt          *time.Time  `gorm:"index"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return
}

type OrderItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name      string    `gorm:"not null"`
	Price     float64   `gorm:"type:decimal(10,2);not null"`
	Quantity  int       `gorm:"not null;check:quantity >= 1"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) (err error) {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return
}
