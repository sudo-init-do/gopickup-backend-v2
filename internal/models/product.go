package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	VendorID      uuid.UUID     `gorm:"type:uuid;not null;index" json:"vendor_id"`
	Vendor        VendorProfile `gorm:"foreignKey:VendorID;references:UserID" json:"vendor,omitempty"` // Association
	Name          string        `gorm:"not null;index" json:"name"` // Indexed for search
	Description   string        `gorm:"type:text" json:"description"`
	Price         float64       `gorm:"type:decimal(10,2);not null;check:price >= 0;index" json:"price"` // Indexed for filtering
	Category      string        `gorm:"not null;index" json:"category"`
	StockQuantity        int           `gorm:"not null;check:stock_quantity >= 0" json:"stock_quantity"`
	Stock                int           `gorm:"-" json:"stock"` // Alias for frontend
	MinimumOrderQuantity int           `gorm:"not null;default:1;check:minimum_order_quantity >= 1" json:"minimum_order_quantity"`
	MOQ                  int           `gorm:"-" json:"moq"` // Alias for frontend
	ImageURL             string        `json:"image_url"`
	IsActive             bool          `gorm:"default:true;index" json:"is_active"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// Ensure database constraints are applied
func (Product) TableName() string {
	return "products"
}

func (p *Product) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}
