package admin

import (
	"github.com/google/uuid"
	"gopickup/internal/models"
	"gopickup/internal/services/admin"
	"gopickup/internal/services/product"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService   *admin.AdminService
	productService *product.ProductService
}

func NewAdminHandler(adminService *admin.AdminService, productService *product.ProductService) *AdminHandler {
	return &AdminHandler{
		adminService:   adminService,
		productService: productService,
	}
}

// GetUsers godoc
// @Summary List users
// @Description List all users on the platform, optionally filter by role
// @Tags admin
// @Accept json
// @Produce json
// @Param role query string false "User Role Filter"
// @Success 200 {array} models.User
// @Failure 500 {object} map[string]string
// @Router /admin/users [get]
func (h *AdminHandler) GetUsers(c *gin.Context) {
	role := c.Query("role")

	users, err := h.adminService.GetUsers(role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetStats godoc
// @Summary Platform stats
// @Description Get platform statistics like user counts and order counts
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} admin.PlatformStats
// @Failure 500 {object} map[string]string
// @Router /admin/stats [get]
func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.adminService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetOrders godoc
// @Summary List all orders
// @Description List all global orders for admin purposes
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {array} models.Order
// @Failure 500 {object} map[string]string
// @Router /admin/orders [get]
func (h *AdminHandler) GetOrders(c *gin.Context) {
	orders, err := h.adminService.GetOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// CreateProduct godoc
// @Summary Admin create product
// @Description Create a product for a specific vendor as an admin
// @Tags admin
// @Accept json
// @Produce json
// @Param request body product.AdminCreateProductRequest true "Product Request"
// @Success 201 {object} models.Product
// @Failure 400 {object} map[string]string
// @Router /admin/products [post]
func (h *AdminHandler) CreateProduct(c *gin.Context) {
	var req product.AdminCreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prod, err := h.productService.CreateProduct(req.VendorID, req.CreateProductRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, prod)
}

func (h *AdminHandler) AssignDriver(c *gin.Context) {
	var body struct {
		OrderID     uuid.UUID `json:"order_id" binding:"required"`
		DriverID    uuid.UUID `json:"driver_id" binding:"required"`
		AgreedPrice float64   `json:"agreed_price" binding:"required"`
		DeliveryFee float64   `json:"delivery_fee" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.adminService.AssignDriver(body.OrderID, body.DriverID, body.AgreedPrice, body.DeliveryFee); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Driver assigned and status updated to assigned"})
}

func (h *AdminHandler) UpdateOrderStatus(c *gin.Context) {
	var body struct {
		OrderID uuid.UUID          `json:"order_id" binding:"required"`
		Status  models.OrderStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.adminService.UpdateOrderStatus(body.OrderID, body.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order status updated by admin"})
}

func (h *AdminHandler) DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.productService.AdminDeleteProduct(productID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted by admin successfully"})
}

