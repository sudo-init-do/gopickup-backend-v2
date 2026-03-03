package driver

import (
	"gopickup/internal/services/driver"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DriverHandler struct {
	service *driver.DriverService
}

func NewDriverHandler(s *driver.DriverService) *DriverHandler {
	return &DriverHandler{service: s}
}

type UpdateLocationRequest struct {
	Lat float64 `json:"lat" binding:"required"`
	Lng float64 `json:"lng" binding:"required"`
}

func (h *DriverHandler) UpdateLocation(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	driverID := userIDInf.(uuid.UUID)

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateLocation(driverID, req.Lat, req.Lng); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "location updated"})
}

func (h *DriverHandler) GetAvailableJobs(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	driverID := userIDInf.(uuid.UUID)

	orders, err := h.service.GetAvailableJobs(driverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

type PlaceBidRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

func (h *DriverHandler) PlaceBid(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	driverID := userIDInf.(uuid.UUID)

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}
	var req PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bid, err := h.service.PlaceBid(driverID, orderID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bid)
}
