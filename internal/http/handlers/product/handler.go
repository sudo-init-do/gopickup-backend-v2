package product

import (
	"gopickup/internal/services/product"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct {
	service *product.ProductService
}

func NewProductHandler(service *product.ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

// Vendor Endpoints

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var req product.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prod, err := h.service.CreateProduct(userID.(uuid.UUID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, prod)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req product.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prod, err := h.service.UpdateProduct(userID.(uuid.UUID), productID, req)
	if err != nil {
		// Could refine error status (403 vs 404 vs 400)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prod)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.service.DeleteProduct(userID.(uuid.UUID), productID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

func (h *ProductHandler) GetVendorDashboard(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	stats, err := h.service.GetVendorDashboard(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ProductHandler) GetMyProducts(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	vendorID := userID.(uuid.UUID)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := product.ProductFilter{
		VendorID: &vendorID,
		Page:     page,
		Limit:    limit,
	}

	resp, err := h.service.ListProducts(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Public Endpoints

func (h *ProductHandler) ListProducts(c *gin.Context) {
	var filter product.ProductFilter

	// Bind query parameters
	if val := c.Query("category"); val != "" {
		filter.Category = &val
	}
	if val := c.Query("min_price"); val != "" {
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			filter.MinPrice = &v
		}
	}
	if val := c.Query("max_price"); val != "" {
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			filter.MaxPrice = &v
		}
	}
	if val := c.Query("vendor_id"); val != "" {
		if v, err := uuid.Parse(val); err == nil {
			filter.VendorID = &v
		}
	}
	if val := c.Query("search"); val != "" {
		filter.Search = &val
	}
	if val := c.Query("page"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			filter.Page = v
		}
	}
	if val := c.Query("limit"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			filter.Limit = v
		}
	}

	resp, err := h.service.ListProducts(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	prod, err := h.service.GetProduct(productID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prod)
}

func (h *ProductHandler) ListVendors(c *gin.Context) {
	var filter product.VendorFilter

	if val := c.Query("business_type"); val != "" {
		filter.BusinessType = &val
	}
	if val := c.Query("search"); val != "" {
		filter.Search = &val
	}
	if val := c.Query("page"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			filter.Page = v
		}
	}
	if val := c.Query("limit"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			filter.Limit = v
		}
	}

	resp, err := h.service.ListVendors(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
