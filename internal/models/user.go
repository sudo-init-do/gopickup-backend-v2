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
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         UserRole  `gorm:"type:varchar(20);not null" json:"role"`
	IsVerified   bool      `gorm:"default:false" json:"is_verified"`
	FCMToken     *string   `gorm:"type:text" json:"fcm_token,omitempty"`

	// OTP fields
	OTPCode      string    `gorm:"type:varchar(6)" json:"-"`
	OTPExpiresAt time.Time `json:"-"`

	// Relations
	ClientProfile *ClientProfile `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"client_profile,omitempty"`
	DriverProfile *DriverProfile `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"driver_profile,omitempty"`
	VendorProfile *VendorProfile `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"vendor_profile,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// BeforeCreate hook to generate UUID if not present (though default:gen_random_uuid() handles it in DB,
// GORM sometimes needs help if we want the ID available immediately after struct creation)
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
