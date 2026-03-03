package models

import (
	"time"

	"github.com/google/uuid"
)

type VehicleType string

const (
	VehicleTricycle VehicleType = "Tricycle"
	VehicleVan      VehicleType = "Van"
	VehicleTruck    VehicleType = "Truck"
	VehicleFlatbed  VehicleType = "Flatbed"
	VehicleTrailer  VehicleType = "Trailer"
)

type ClientProfile struct {
	UserID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName          string    `gorm:"not null"`
	PhoneNumber       string    `gorm:"unique;not null"`
	Address           string    `gorm:"not null"`
	ProfilePictureURL *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type DriverProfile struct {
	UserID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName          string    `gorm:"not null"`
	PhoneNumber       string    `gorm:"unique;not null"`
	LicenseNumber     string    `gorm:"unique;not null"`
	VehicleType       VehicleType `gorm:"type:varchar(50);not null"`
	PlateNumber       string    `gorm:"unique;not null"`
	VehicleCapacity   float64   `gorm:"not null"` // Assuming capacity is a number (e.g., tons or kg)
	IsApproved        bool      `gorm:"default:false"`
	CurrentLocationLat *float64
	CurrentLocationLng *float64
	// Location field (PostGIS) omitted for now to keep simple compatibility with SQLite for tests, 
	// but can be added if Postgres is strictly enforced.
	// We will rely on Lat/Lng for API logic for now.
	
	ProfilePictureURL *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type VendorProfile struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	StoreName      string    `gorm:"not null"`
	PhoneNumber    string    `gorm:"unique;not null"`
	BusinessType   string    `gorm:"not null"`
	Address        string    `gorm:"not null"`
	StoreBannerURL *string
	IsApproved     bool      `gorm:"default:false"`
	Products       []Product `gorm:"foreignKey:VendorID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
