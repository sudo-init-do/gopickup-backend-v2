package admin

import (
	"gopickup/internal/services/admin"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service *admin.AdminService
}

func NewAdminHandler(service *admin.AdminService) *AdminHandler {
	return &AdminHandler{
		service: service,
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

	users, err := h.service.GetUsers(role)
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
	stats, err := h.service.GetStats()
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
	orders, err := h.service.GetOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}
