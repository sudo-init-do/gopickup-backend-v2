package profile

import (
	"errors"
	"fmt"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/email"

	"github.com/google/uuid"
)

type ProfileService struct {
	emailService email.EmailService
}

func NewProfileService(emailService email.EmailService) *ProfileService {
	return &ProfileService{
		emailService: emailService,
	}
}

// Request structs
type CreateClientProfileRequest struct {
	FullName          string  `json:"full_name" binding:"required"`
	PhoneNumber       string  `json:"phone_number" binding:"required"`
	Address           string  `json:"address" binding:"required"`
	ProfilePictureURL *string `json:"profile_picture_url"`
}

type CreateDriverProfileRequest struct {
	FullName          string             `json:"full_name" binding:"required"`
	PhoneNumber       string             `json:"phone_number" binding:"required"`
	LicenseNumber     string             `json:"license_number" binding:"required"`
	VehicleType       models.VehicleType `json:"vehicle_type" binding:"required,oneof=Tricycle Van Truck Flatbed Trailer"`
	PlateNumber       string             `json:"plate_number" binding:"required"`
	VehicleCapacity   float64            `json:"vehicle_capacity" binding:"required"`
	ProfilePictureURL *string            `json:"profile_picture_url"`
}

type CreateVendorProfileRequest struct {
	StoreName      string  `json:"store_name" binding:"required"`
	PhoneNumber    string  `json:"phone_number" binding:"required"`
	BusinessType   string  `json:"business_type" binding:"required"`
	Address        string  `json:"address" binding:"required"`
	StoreBannerURL *string `json:"store_banner_url"`
}

type UpdateProfileRequest struct {
	// Common fields
	FullName          *string `json:"full_name"`
	PhoneNumber       *string `json:"phone_number"`
	Address           *string `json:"address"`
	ProfilePictureURL *string `json:"profile_picture_url"`

	// Driver specific
	VehicleType     *models.VehicleType `json:"vehicle_type"`
	VehicleCapacity *float64            `json:"vehicle_capacity"`
	// Note: LicenseNumber and PlateNumber are usually unique/verified and might not be editable freely,
	// but user said "update only allowed fields". I'll allow them for now but they must remain unique.

	// Vendor specific
	StoreName      *string `json:"store_name"`
	BusinessType   *string `json:"business_type"`
	StoreBannerURL *string `json:"store_banner_url"`
}

// Create methods
func (s *ProfileService) CreateClientProfile(userID uuid.UUID, req CreateClientProfileRequest) error {
	// Check if user is actually a client
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return err
	}
	if user.Role != models.RoleClient {
		return errors.New("user is not a client")
	}

	// Check if profile exists
	var existing models.ClientProfile
	if err := db.DB.First(&existing, userID).Error; err == nil {
		return errors.New("profile already exists")
	}

	profile := models.ClientProfile{
		UserID:            userID,
		FullName:          req.FullName,
		PhoneNumber:       req.PhoneNumber,
		Address:           req.Address,
		ProfilePictureURL: req.ProfilePictureURL,
	}

	return db.DB.Create(&profile).Error
}

func (s *ProfileService) CreateDriverProfile(userID uuid.UUID, req CreateDriverProfileRequest) error {
	// Check if user is actually a driver
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return err
	}
	if user.Role != models.RoleDriver {
		return errors.New("user is not a driver")
	}

	// Check if profile exists
	var existing models.DriverProfile
	if err := db.DB.First(&existing, userID).Error; err == nil {
		return errors.New("profile already exists")
	}

	profile := models.DriverProfile{
		UserID:            userID,
		FullName:          req.FullName,
		PhoneNumber:       req.PhoneNumber,
		LicenseNumber:     req.LicenseNumber,
		VehicleType:       req.VehicleType,
		PlateNumber:       req.PlateNumber,
		VehicleCapacity:   req.VehicleCapacity,
		IsApproved:        false, // Default
		ProfilePictureURL: req.ProfilePictureURL,
	}

	return db.DB.Create(&profile).Error
}

func (s *ProfileService) CreateVendorProfile(userID uuid.UUID, req CreateVendorProfileRequest) error {
	// Check if user is actually a vendor
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return err
	}
	if user.Role != models.RoleVendor {
		return errors.New("user is not a vendor")
	}

	// Check if profile exists
	var existing models.VendorProfile
	if err := db.DB.First(&existing, userID).Error; err == nil {
		return errors.New("profile already exists")
	}

	profile := models.VendorProfile{
		UserID:         userID,
		StoreName:      req.StoreName,
		PhoneNumber:    req.PhoneNumber,
		BusinessType:   req.BusinessType,
		Address:        req.Address,
		StoreBannerURL: req.StoreBannerURL,
		IsApproved:     false, // Default
	}

	return db.DB.Create(&profile).Error
}

// Update method
func (s *ProfileService) UpdateProfile(userID uuid.UUID, role models.UserRole, req UpdateProfileRequest) error {
	switch role {
	case models.RoleClient:
		var profile models.ClientProfile
		if err := db.DB.First(&profile, userID).Error; err != nil {
			return errors.New("profile not found")
		}
		if req.FullName != nil {
			profile.FullName = *req.FullName
		}
		if req.PhoneNumber != nil {
			profile.PhoneNumber = *req.PhoneNumber
		}
		if req.Address != nil {
			profile.Address = *req.Address
		}
		if req.ProfilePictureURL != nil {
			profile.ProfilePictureURL = req.ProfilePictureURL
		}
		return db.DB.Save(&profile).Error

	case models.RoleDriver:
		var profile models.DriverProfile
		if err := db.DB.First(&profile, userID).Error; err != nil {
			return errors.New("profile not found")
		}
		if req.FullName != nil {
			profile.FullName = *req.FullName
		}
		if req.PhoneNumber != nil {
			profile.PhoneNumber = *req.PhoneNumber
		}
		// LicenseNumber and PlateNumber usually require re-verification if changed, but allow for now if user wants
		if req.VehicleType != nil {
			profile.VehicleType = *req.VehicleType
		}
		if req.VehicleCapacity != nil {
			profile.VehicleCapacity = *req.VehicleCapacity
		}
		if req.ProfilePictureURL != nil {
			profile.ProfilePictureURL = req.ProfilePictureURL
		}
		return db.DB.Save(&profile).Error

	case models.RoleVendor:
		var profile models.VendorProfile
		if err := db.DB.First(&profile, userID).Error; err != nil {
			return errors.New("profile not found")
		}
		if req.StoreName != nil {
			profile.StoreName = *req.StoreName
		}
		if req.PhoneNumber != nil {
			profile.PhoneNumber = *req.PhoneNumber
		}
		if req.BusinessType != nil {
			profile.BusinessType = *req.BusinessType
		}
		if req.Address != nil {
			profile.Address = *req.Address
		}
		if req.StoreBannerURL != nil {
			profile.StoreBannerURL = req.StoreBannerURL
		}
		return db.DB.Save(&profile).Error

	default:
		return errors.New("invalid role for profile update")
	}
}

// Admin methods
func (s *ProfileService) ApproveDriver(driverID uuid.UUID) error {
	var profile models.DriverProfile
	if err := db.DB.First(&profile, driverID).Error; err != nil {
		return errors.New("driver profile not found")
	}

	if profile.IsApproved {
		return nil // Already approved
	}

	profile.IsApproved = true
	if err := db.DB.Save(&profile).Error; err != nil {
		return err
	}

	// Send notification email
	var user models.User
	if err := db.DB.First(&user, driverID).Error; err == nil {
		go func() {
			if err := s.emailService.SendEmail(
				user.Email,
				"Driver Account Approved",
				getApprovalEmailTemplate("Driver"),
				"Congratulations! Your driver account has been approved. You can now start accepting jobs.",
			); err != nil {
				fmt.Printf("Failed to send approval email: %v\n", err)
			}
		}()
	}

	return nil
}

func (s *ProfileService) ApproveVendor(vendorID uuid.UUID) error {
	var profile models.VendorProfile
	if err := db.DB.First(&profile, vendorID).Error; err != nil {
		return errors.New("vendor profile not found")
	}

	if profile.IsApproved {
		return nil // Already approved
	}

	profile.IsApproved = true
	if err := db.DB.Save(&profile).Error; err != nil {
		return err
	}

	// Send notification email
	var user models.User
	if err := db.DB.First(&user, vendorID).Error; err == nil {
		go func() {
			if err := s.emailService.SendEmail(
				user.Email,
				"Vendor Account Approved",
				getApprovalEmailTemplate("Vendor"),
				"Congratulations! Your vendor account has been approved. You can now start listing products.",
			); err != nil {
				fmt.Printf("Failed to send approval email: %v\n", err)
			}
		}()
	}

	return nil
}

func getApprovalEmailTemplate(role string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f6f9fc; margin: 0; padding: 0; color: #333333; }
        .wrapper { width: 100%%; background-color: #f6f9fc; padding: 40px 0; }
        .container { max-width: 460px; margin: 0 auto; background-color: #ffffff; border-radius: 16px; box-shadow: 0 4px 20px rgba(0,0,0,0.05); overflow: hidden; }
        .header { padding: 40px 40px 20px 40px; text-align: center; }
        .logo { font-size: 24px; font-weight: 800; color: #111111; letter-spacing: -0.5px; text-decoration: none; }
        .content { padding: 0 40px 40px 40px; text-align: center; }
        .title { font-size: 20px; font-weight: 600; margin-bottom: 16px; color: #1a1a1a; }
        .text { font-size: 15px; line-height: 1.6; color: #4e5c6e; margin-bottom: 32px; }
        .status-container { background-color: #f0f4f8; border-radius: 12px; padding: 24px; margin-bottom: 32px; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; font-size: 24px; font-weight: 700; color: #00cc66; letter-spacing: 1px; text-transform: uppercase; }
        .footer { padding: 20px; text-align: center; color: #8898aa; font-size: 12px; line-height: 1.5; }
        .footer a { color: #8898aa; text-decoration: none; border-bottom: 1px solid #dce4ec; }
    </style>
</head>
<body>
    <div class="wrapper">
        <div class="container">
            <div class="header">
                <div class="logo">GoPickup</div>
            </div>
            <div class="content">
                <h1 class="title">You're in!</h1>
                <p class="text">
                    Your application to join GoPickup as a <strong>%s</strong> has been approved. You are now ready to start.
                </p>
                
                <div class="status-container">
                    Account Approved
                </div>
            </div>
        </div>
        <div class="footer">
            &copy; 2026 GoPickup Inc.<br>
            Sent with &hearts; from GoPickup Team
        </div>
    </div>
</body>
</html>`, role)
}

// Get Profile Helpers (optional, but good for returning data)
func (s *ProfileService) GetProfile(userID uuid.UUID, role models.UserRole) (interface{}, error) {
	switch role {
	case models.RoleClient:
		var p models.ClientProfile
		err := db.DB.First(&p, userID).Error
		return p, err
	case models.RoleDriver:
		var p models.DriverProfile
		err := db.DB.First(&p, userID).Error
		return p, err
	case models.RoleVendor:
		var p models.VendorProfile
		err := db.DB.First(&p, userID).Error
		return p, err
	default:
		return nil, errors.New("unknown role")
	}
}
