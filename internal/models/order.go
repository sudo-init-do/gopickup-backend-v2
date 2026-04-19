package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderPending         OrderStatus = "pending"           // Order created, negotiation not yet started
	OrderAwaitingPayment OrderStatus = "awaiting_payment"  // Client redirected to WhatsApp to negotiate
	OrderPaymentMade     OrderStatus = "payment_made"      // Client self-reported payment, awaiting admin verification
	OrderProcessing      OrderStatus = "processing"        // Admin verified payment; visible to all approved drivers
	OrderAssigned        OrderStatus = "assigned"          // Admin manually assigned a specific driver (override)
	OrderInProgress      OrderStatus = "in_progress"       // Driver tapped "I Accept This Job"
	OrderPickedUp        OrderStatus = "picked_up"
	OrderOnTheWay        OrderStatus = "on_the_way"
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
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	DriverID  uuid.UUID `gorm:"type:uuid;not null;index" json:"driver_id"`
	Amount    float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status    BidStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
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
	PaymentWhatsApp       PaymentMethod = "whatsapp"
)

type Order struct {
	ID                 uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	ClientID           uuid.UUID     `gorm:"type:uuid;not null;index" json:"client_id"`
	VendorID           uuid.UUID     `gorm:"type:uuid;not null;index" json:"vendor_id"`
	DriverID           *uuid.UUID    `gorm:"type:uuid;index" json:"driver_id,omitempty"`
	TotalProductAmount float64       `gorm:"type:decimal(10,2);not null" json:"total_product_amount"`
	AgreedPrice        *float64      `gorm:"type:decimal(10,2)" json:"agreed_price,omitempty"`     // Total after negotiation
	AgreedDeliveryFee  *float64      `gorm:"type:decimal(10,2)" json:"agreed_delivery_fee,omitempty"`
	PaymentMethod      PaymentMethod `gorm:"type:varchar(30);not null" json:"payment_method"`
	PickupAddress      string        `gorm:"not null" json:"pickup_address"`
	DeliveryAddress    string        `gorm:"not null" json:"delivery_address"`
	DeliveryLat        *float64      `json:"delivery_lat"`
	DeliveryLng        *float64      `json:"delivery_lng"`
	Status             OrderStatus   `gorm:"type:varchar(30);not null;default:'pending';index" json:"status"`
	
	Client             *User         `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	Vendor             VendorProfile `gorm:"foreignKey:VendorID" json:"vendor,omitempty"`
	Items []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Bids  []Bid       `gorm:"foreignKey:OrderID" json:"bids,omitempty"`

	CreatedAt time.Time  `gorm:"index" json:"created_at"` // Indexed for sorting
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return
}

type OrderItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	Name      string    `gorm:"not null" json:"name"`
	Price     float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Quantity  int       `gorm:"not null;check:quantity >= 1" json:"quantity"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) (err error) {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return
}
