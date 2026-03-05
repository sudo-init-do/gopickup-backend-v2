package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey"`
	VendorID      uuid.UUID     `gorm:"type:uuid;not null;index"`
	Vendor        VendorProfile `gorm:"foreignKey:VendorID;references:UserID"` // Association
	Name          string        `gorm:"not null"`
	Description   string        `gorm:"type:text"`
	Price         float64       `gorm:"type:decimal(10,2);not null;check:price >= 0"`
	Category      string        `gorm:"not null;index"`
	StockQuantity int           `gorm:"not null;check:stock_quantity >= 0"`
	ImageURL      string
	IsActive      bool `gorm:"default:true;index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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
