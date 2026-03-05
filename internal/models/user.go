package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleClient UserRole = "client"
	RoleDriver UserRole = "driver"
	RoleVendor UserRole = "vendor"
	RoleAdmin  UserRole = "admin"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         UserRole  `gorm:"type:varchar(20);not null"`
	IsVerified   bool      `gorm:"default:false"`
	FCMToken     *string   `gorm:"type:text"`

	// OTP fields
	OTPCode      string    `gorm:"type:varchar(6)" json:"-"`
	OTPExpiresAt time.Time `json:"-"`

	// Relations
	ClientProfile *ClientProfile `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	DriverProfile *DriverProfile `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	VendorProfile *VendorProfile `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// BeforeCreate hook to generate UUID if not present (though default:gen_random_uuid() handles it in DB,
// GORM sometimes needs help if we want the ID available immediately after struct creation)
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
