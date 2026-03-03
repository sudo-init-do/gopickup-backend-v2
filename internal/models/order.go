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
	OrderCancelled       OrderStatus = "cancelled"
)

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

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) (err error) {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return
}
