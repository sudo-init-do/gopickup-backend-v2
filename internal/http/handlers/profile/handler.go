package profile

import (
	"gopickup/internal/models"
	"gopickup/internal/services/profile"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProfileHandler struct {
	service *profile.ProfileService
}

func NewProfileHandler(service *profile.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		service: service,
	}
}

// CreateClient godoc
// @Summary Create client profile
// @Description Create a new client profile for the authenticated user
// @Tags profile
// @Accept json
// @Produce json
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /profile/client [post]
func (h *ProfileHandler) CreateClient(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var req profile.CreateClientProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateClientProfile(userID.(uuid.UUID), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Client profile created successfully"})
}

// CreateDriver godoc
// @Summary Create driver profile
// @Description Create a new driver profile for the authenticated user
// @Tags profile
// @Accept json
// @Produce json
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /profile/driver [post]
func (h *ProfileHandler) CreateDriver(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var req profile.CreateDriverProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateDriverProfile(userID.(uuid.UUID), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Driver profile created successfully"})
}

// CreateVendor godoc
// @Summary Create vendor profile
// @Description Create a new vendor profile for the authenticated user
// @Tags profile
// @Accept json
// @Produce json
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /profile/vendor [post]
func (h *ProfileHandler) CreateVendor(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var req profile.CreateVendorProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateVendorProfile(userID.(uuid.UUID), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Vendor profile created successfully"})
}

// UpdateProfile godoc
// @Summary Update profile
// @Description Update the profile of the authenticated user based on their role
// @Tags profile
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /profile [put]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Role not found in context"})
		return
	}

	var req profile.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateProfile(userID.(uuid.UUID), models.UserRole(role.(string)), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get the profile details of the authenticated user based on their role
// @Tags profile
// @Produce json
// @Success 200 {object} interface{}
// @Failure 404 {object} map[string]string
// @Router /profile [get]
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Role not found in context"})
		return
	}

	profile, err := h.service.GetProfile(userID.(uuid.UUID), models.UserRole(role.(string)))
	if err != nil {
		// Instead of 404, return a 200 with a flag that profile is missing.
		// This prevents frontend "404 Not Found" crashes.
		c.JSON(http.StatusOK, gin.H{
			"user_id":            userID,
			"role":               role,
			"is_profile_created": false,
			"message":            "Profile data is currently empty. Please create your profile.",
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ApproveDriver godoc
// @Summary Approve driver
// @Description Approve a driver account (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /admin/drivers/{user_id}/approve [patch]
func (h *ProfileHandler) ApproveDriver(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.service.ApproveDriver(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Driver approved successfully"})
}

// ApproveVendor godoc
// @Summary Approve vendor
// @Description Approve a vendor account (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /admin/vendors/{user_id}/approve [patch]
func (h *ProfileHandler) ApproveVendor(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.service.ApproveVendor(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vendor approved successfully"})
}

// DeleteAccount godoc
// @Summary Delete user account
// @Description Permanently delete the authenticated user's account and all associated data
// @Tags profile
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /profile [delete]
func (h *ProfileHandler) DeleteAccount(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	if err := h.service.DeleteAccount(userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account successfully deleted"})
}

// DeleteAccountPage returns a simple HTML landing page for Google Play compliance
func (h *ProfileHandler) DeleteAccountPage(c *gin.Context) {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>GoPickup - Account Deletion</title>
		<style>
			body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background-color: #f4f7f6; }
			.card { background: white; padding: 40px; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; max-width: 400px; }
			h1 { color: #333; }
			p { color: #666; line-height: 1.6; }
			.brand { color: #5B21B6; font-weight: bold; }
		</style>
	</head>
	<body>
		<div class="card">
			<h1 class="brand">GoPickup</h1>
			<h2>Account Deletion</h2>
			<p>To delete your account and all associated data, please follow these steps inside the <b>GoPickup App</b>:</p>
			<p>1. Open the App<br>2. Go to <b>Profile Settings</b><br>3. Tap <b>Delete Account</b></p>
			<p>Your data will be permanently removed from our systems within 24 hours of your request.</p>
		</div>
	</body>
	</html>
	`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}


