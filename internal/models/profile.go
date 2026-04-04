package models

import (
	"time"

	"github.com/google/uuid"
)

type VehicleType string

const (
	VehicleCanter    VehicleType = "Canter"
	VehicleTipper    VehicleType = "Tipper"
	VehicleTricycle  VehicleType = "Tricycle"
	VehicleVan       VehicleType = "Van"
	VehicleTruck     VehicleType = "Truck"
	VehicleTrucks    VehicleType = "Trucks"
	VehicleFlatbed   VehicleType = "Flatbed"
	VehicleFlatbeds  VehicleType = "Flatbeds"
	VehicleTrailer   VehicleType = "Trailer"
)

type ClientProfile struct {
	UserID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	FullName          string    `gorm:"not null" json:"full_name"`
	PhoneNumber       string    `gorm:"not null" json:"phone_number"`
	Address           string    `gorm:"not null" json:"address"`
	ProfilePictureURL *string   `json:"profile_picture_url,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DriverProfile struct {
	UserID             uuid.UUID   `gorm:"type:uuid;primaryKey" json:"user_id"`
	FullName           string      `gorm:"not null" json:"full_name"`
	PhoneNumber        string      `gorm:"not null" json:"phone_number"`
	LicenseNumber      string      `gorm:"unique;not null" json:"license_number"`
	VehicleType        VehicleType `gorm:"type:varchar(50);not null" json:"vehicle_type"`
	PlateNumber        string      `gorm:"unique;not null" json:"plate_number"`
	VehicleCapacity    float64     `gorm:"not null" json:"vehicle_capacity"` // Assuming capacity is a number (e.g., tons or kg)
	IsApproved         bool        `gorm:"default:false" json:"is_approved"`
	CurrentLocationLat *float64    `json:"current_location_lat,omitempty"`
	CurrentLocationLng *float64    `json:"current_location_lng,omitempty"`
	// Location field (PostGIS) omitted for now to keep simple compatibility with SQLite for tests,
	// but can be added if Postgres is strictly enforced.
	// We will rely on Lat/Lng for API logic for now.

	ProfilePictureURL *string `json:"profile_picture_url,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VendorProfile struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	StoreName                  string    `gorm:"not null" json:"store_name"`
	PhoneNumber                string    `gorm:"not null" json:"phone_number"`
	BusinessType               string    `gorm:"not null" json:"business_type"`
	Address                    string    `gorm:"not null" json:"address"`
	BusinessRegistrationNumber string    `json:"business_registration_number"`
	StoreBannerURL             *string   `json:"store_banner_url,omitempty"`
	IsApproved     bool      `gorm:"default:false" json:"is_approved"`
	Products       []Product `gorm:"foreignKey:VendorID" json:"products,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
