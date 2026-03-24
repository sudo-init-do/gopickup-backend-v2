package load

import (
	"gopickup/internal/models"
	loadSvc "gopickup/internal/services/load"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LoadHandler struct {
	service *loadSvc.LoadService
}

func NewLoadHandler(service *loadSvc.LoadService) *LoadHandler {
	return &LoadHandler{service: service}
}

// --- Client Handlers ---

// CreateLoad godoc
// POST /loads
func (h *LoadHandler) CreateLoad(c *gin.Context) {
	clientID := c.MustGet("userID").(uuid.UUID)

	var req loadSvc.CreateLoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	load, err := h.service.CreateLoad(clientID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, load)
}

// ListMyLoads godoc
// GET /loads/my
func (h *LoadHandler) ListMyLoads(c *gin.Context) {
	clientID := c.MustGet("userID").(uuid.UUID)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	loads, count, err := h.service.ListClientLoads(clientID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": loads, "total": count})
}

// GetLoad godoc
// GET /loads/:id
func (h *LoadHandler) GetLoad(c *gin.Context) {
	clientID := c.MustGet("userID").(uuid.UUID)
	loadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid load ID"})
		return
	}

	load, err := h.service.GetLoad(clientID, loadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, load)
}

// AcceptLoadBid godoc
// POST /loads/:id/bids/:bid_id/accept
func (h *LoadHandler) AcceptLoadBid(c *gin.Context) {
	clientID := c.MustGet("userID").(uuid.UUID)
	loadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid load ID"})
		return
	}
	bidID, err := uuid.Parse(c.Param("bid_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bid ID"})
		return
	}

	load, err := h.service.AcceptLoadBid(clientID, loadID, bidID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, load)
}

// CancelLoad godoc
// PATCH /loads/:id/cancel
func (h *LoadHandler) CancelLoad(c *gin.Context) {
	clientID := c.MustGet("userID").(uuid.UUID)
	loadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid load ID"})
		return
	}

	load, err := h.service.CancelLoad(clientID, loadID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, load)
}

// --- Driver Handlers ---

// ListAvailableLoads godoc
// GET /loads/available
func (h *LoadHandler) ListAvailableLoads(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	loads, count, err := h.service.ListAvailableLoads(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": loads, "total": count})
}

// PlaceLoadBid godoc
// POST /loads/:id/bid
func (h *LoadHandler) PlaceLoadBid(c *gin.Context) {
	driverID := c.MustGet("userID").(uuid.UUID)
	loadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid load ID"})
		return
	}

	var req loadSvc.PlaceLoadBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bid, err := h.service.PlaceLoadBid(driverID, loadID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bid)
}

// UpdateLoadStatus godoc
// PATCH /loads/:id/status
func (h *LoadHandler) UpdateLoadStatus(c *gin.Context) {
	driverID := c.MustGet("userID").(uuid.UUID)
	loadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid load ID"})
		return
	}

	var body struct {
		Status models.LoadStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	load, err := h.service.UpdateLoadStatus(driverID, loadID, body.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, load)
}
