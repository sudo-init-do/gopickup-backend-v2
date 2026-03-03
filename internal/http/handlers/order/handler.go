package order

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopickup/internal/models"
	"gopickup/internal/services/order"
)

type OrderHandler struct {
	service *order.OrderService
}

func NewOrderHandler(s *order.OrderService) *OrderHandler {
	return &OrderHandler{service: s}
}

func (h *OrderHandler) Checkout(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	roleInf, _ := c.Get("role")
	if roleInf == nil || roleInf.(string) != string(models.RoleClient) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only clients can checkout"})
		return
	}
	var req order.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	o, err := h.service.Checkout(userIDInf.(uuid.UUID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, o)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	roleInf, _ := c.Get("role")
	role := models.UserRole(roleInf.(string))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	orders, total, err := h.service.ListOrders(userIDInf.(uuid.UUID), role, page, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	totalPages := (int(total) + limit - 1) / limit
	c.JSON(http.StatusOK, gin.H{
		"data": orders,
		"meta": gin.H{
			"current_page": page,
			"total_pages":  totalPages,
			"total_items":  total,
			"limit":        limit,
		},
	})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	roleInf, _ := c.Get("role")
	role := models.UserRole(roleInf.(string))
	idStr := c.Param("id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	o, err := h.service.GetOrder(userIDInf.(uuid.UUID), role, orderID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *OrderHandler) VendorUpdateStatus(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	roleInf, _ := c.Get("role")
	if roleInf == nil || roleInf.(string) != string(models.RoleVendor) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only vendor can update status"})
		return
	}
	idStr := c.Param("id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Status models.OrderStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	o, err := h.service.VendorUpdateStatus(userIDInf.(uuid.UUID), orderID, body.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *OrderHandler) VendorMarkReady(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	roleInf, _ := c.Get("role")
	if roleInf == nil || roleInf.(string) != string(models.RoleVendor) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only vendor can mark ready"})
		return
	}
	idStr := c.Param("id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	o, err := h.service.VendorMarkReady(userIDInf.(uuid.UUID), orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}
