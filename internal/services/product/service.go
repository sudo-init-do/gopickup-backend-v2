package product

import (
	"context"
	"encoding/json"
	"errors"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/audit"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ProductService struct {
	audit  *audit.AuditService
	redis  *redis.Client
	config *config.Config
}

func NewProductService(audit *audit.AuditService, redis *redis.Client, cfg *config.Config) *ProductService {
	return &ProductService{audit: audit, redis: redis, config: cfg}
}

// DTOs

type CreateProductRequest struct {
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description"`
	Price         float64 `json:"price" binding:"required,min=0"`
	Category      string  `json:"category" binding:"required"`
	StockQuantity        int     `json:"stock_quantity"`
	Stock                int     `json:"stock"`                 // Alias for stock_quantity
	MinimumOrderQuantity int     `json:"minimum_order_quantity"`
	MOQ                  int     `json:"moq"`                   // Alias for minimum_order_quantity
	ImageURL             string  `json:"image_url"`
}

type AdminCreateProductRequest struct {
	VendorID uuid.UUID `json:"vendor_id" binding:"required"`
	CreateProductRequest
}

type UpdateProductRequest struct {
	Name          *string  `json:"name"`
	Description   *string  `json:"description"`
	Price         *float64 `json:"price" binding:"omitempty,min=0"`
	Category      *string  `json:"category"`
	StockQuantity        *int     `json:"stock_quantity" binding:"omitempty,min=0"`
	Stock                *int     `json:"stock" binding:"omitempty,min=0"`
	MinimumOrderQuantity *int     `json:"minimum_order_quantity" binding:"omitempty,min=1"`
	MOQ                  *int     `json:"moq" binding:"omitempty,min=1"`
	ImageURL             *string  `json:"image_url"`
	IsActive      *bool    `json:"is_active"`
}

type ProductFilter struct {
	Category      *string
	MinPrice      *float64
	MaxPrice      *float64
	VendorID      *uuid.UUID
	Search        *string
	Page          int
	Limit         int
	UniqueVendors bool // If true, only return one product per vendor (latest)
}

type VendorFilter struct {
	BusinessType *string
	Search       *string
	Page         int
	Limit        int
}

type PaginationMeta struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	TotalItems  int `json:"total_items"`
	Limit       int `json:"limit"`
}

type PaginatedResponse struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// Vendor Methods

func (s *ProductService) CreateProduct(vendorID uuid.UUID, req CreateProductRequest) (*models.Product, error) {
	// Handle Aliases
	if req.StockQuantity == 0 && req.Stock > 0 {
		req.StockQuantity = req.Stock
	}
	if req.MinimumOrderQuantity == 0 && req.MOQ > 0 {
		req.MinimumOrderQuantity = req.MOQ
	}
	
	// 1. Check if vendor is approved
	var vendor models.VendorProfile
	if err := db.DB.First(&vendor, vendorID).Error; err != nil {
		return nil, errors.New("vendor profile not found")
	}
	if !vendor.IsApproved {
		return nil, errors.New("vendor is not approved")
	}

	// 2. Create product
	product := models.Product{
		VendorID:      vendorID,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		Category:      req.Category,
		StockQuantity:        req.StockQuantity,
		MinimumOrderQuantity: req.MinimumOrderQuantity,
		ImageURL:             req.ImageURL,
		IsActive:             true,
	}

	if err := db.DB.Create(&product).Error; err != nil {
		return nil, err
	}

	s.audit.Log(vendorID, "PRODUCT_CREATED", "product", product.ID, nil)

	return &product, nil
}

func (s *ProductService) UpdateProduct(vendorID uuid.UUID, productID uuid.UUID, req UpdateProductRequest) (*models.Product, error) {
	// 1. Check if vendor is approved (optional but good practice)
	var vendor models.VendorProfile
	if err := db.DB.First(&vendor, vendorID).Error; err != nil {
		return nil, errors.New("vendor profile not found")
	}
	if !vendor.IsApproved {
		return nil, errors.New("vendor is not approved")
	}

	// 2. Find product and check ownership
	var product models.Product
	if err := db.DB.First(&product, productID).Error; err != nil {
		return nil, errors.New("product not found")
	}

	if product.VendorID != vendorID {
		return nil, errors.New("unauthorized: you do not own this product")
	}

	// 3. Update fields
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Category != nil {
		product.Category = *req.Category
	}
	if req.StockQuantity != nil {
		product.StockQuantity = *req.StockQuantity
	}
	if req.ImageURL != nil {
		product.ImageURL = *req.ImageURL
	}
	if req.MinimumOrderQuantity != nil {
		product.MinimumOrderQuantity = *req.MinimumOrderQuantity
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	if err := db.DB.Save(&product).Error; err != nil {
		return nil, err
	}

	s.audit.Log(vendorID, "PRODUCT_UPDATED", "product", product.ID, nil)

	// Invalidate cache
	if s.redis != nil {
		s.redis.Del(context.Background(), "product:"+product.ID.String())
	}

	return &product, nil
}

func (s *ProductService) DeleteProduct(vendorID uuid.UUID, productID uuid.UUID) error {
	// 1. Check ownership
	var product models.Product
	if err := db.DB.First(&product, productID).Error; err != nil {
		return errors.New("product not found")
	}

	if product.VendorID != vendorID {
		return errors.New("unauthorized: you do not own this product")
	}

	// 2. Soft delete (set IsActive = false) per requirements
	// The requirement says "soft delete preferred: set is_active=false"
	// However, GORM has soft delete support via DeletedAt.
	// The user explicitly asked to set is_active=false.
	// I will do both or just is_active=false. Let's do is_active=false to match requirements exactly.

	product.IsActive = false
	if err := db.DB.Save(&product).Error; err != nil {
		return err
	}

	err := db.DB.Delete(&product).Error
	if err == nil {
		s.audit.Log(vendorID, "PRODUCT_DELETED", "product", product.ID, nil)
		if s.redis != nil {
			s.redis.Del(context.Background(), "product:"+product.ID.String())
		}
	}
	return err
}

func (s *ProductService) AdminDeleteProduct(productID uuid.UUID) error {
	var product models.Product
	if err := db.DB.First(&product, productID).Error; err != nil {
		return errors.New("product not found")
	}

	// Hard delete the product using an explicit WHERE clause 
	// Gorm silently fails on Delete(&struct) sometimes with custom UUID types
	err := db.DB.Unscoped().Where("id = ?", productID).Delete(&models.Product{}).Error
	if err == nil {
		// Log system delete
		s.audit.Log(uuid.Nil, "PRODUCT_HARD_DELETED_BY_ADMIN", "product", product.ID, nil)
		if s.redis != nil {
			s.redis.Del(context.Background(), "product:"+product.ID.String())
		}
	}
	return err
}


type VendorDashboardStats struct {
	TotalProducts  int64   `json:"total_products"`
	ActiveProducts int64   `json:"active_products"`
	TotalOrders    int64   `json:"total_orders"`
	PendingOrders  int64   `json:"pending_orders"`
	TotalRevenue   float64 `json:"total_revenue"`
}

func (s *ProductService) GetVendorDashboard(vendorID uuid.UUID) (*VendorDashboardStats, error) {
	var stats VendorDashboardStats

	// 1. Product Stats
	if err := db.DB.Model(&models.Product{}).Where("vendor_id = ?", vendorID).Count(&stats.TotalProducts).Error; err != nil {
		return nil, err
	}
	if err := db.DB.Model(&models.Product{}).Where("vendor_id = ? AND is_active = ?", vendorID, true).Count(&stats.ActiveProducts).Error; err != nil {
		return nil, err
	}

	// 2. Order Stats
	if err := db.DB.Model(&models.Order{}).Where("vendor_id = ?", vendorID).Count(&stats.TotalOrders).Error; err != nil {
		return nil, err
	}
	// Pending orders (Pending, Processing, Assigned, InProgress)
	pendingStatuses := []models.OrderStatus{
		models.OrderPending,
		models.OrderProcessing,
		models.OrderAssigned,
		models.OrderInProgress,
	}
	if err := db.DB.Model(&models.Order{}).Where("vendor_id = ? AND status IN ?", vendorID, pendingStatuses).Count(&stats.PendingOrders).Error; err != nil {
		return nil, err
	}

	// 3. Revenue Stats (Only from non-cancelled orders)
	if err := db.DB.Model(&models.Order{}).
		Where("vendor_id = ? AND status != ?", vendorID, models.OrderCancelled).
		Select("COALESCE(SUM(total_product_amount), 0)").
		Row().Scan(&stats.TotalRevenue); err != nil {
		log.Printf("Error calculating revenue: %v", err)
	}

	return &stats, nil
}

// Public Methods

func (s *ProductService) ListProducts(filter ProductFilter) (*PaginatedResponse, error) {
	if filter.VendorID != nil {
		log.Printf("DEBUG: Listing products for Vendor: %s", filter.VendorID.String())
	}

	var products []models.Product
	var totalItems int64

	query := db.DB.Model(&models.Product{}).Where("is_active = ?", true)

	// Filters
	if filter.Category != nil && *filter.Category != "" {
		query = query.Where("category = ?", *filter.Category)
	}
	if filter.MinPrice != nil {
		query = query.Where("price >= ?", *filter.MinPrice)
	}
	if filter.MaxPrice != nil {
		query = query.Where("price <= ?", *filter.MaxPrice)
	}
	if filter.VendorID != nil {
		query = query.Where("vendor_id = ?", *filter.VendorID)
	}
	if filter.Search != nil && *filter.Search != "" {
		searchTerm := "%" + *filter.Search + "%"
		// Join with vendor profile to search by store name too
		query = query.Joins("JOIN vendor_profiles ON vendor_profiles.user_id = products.vendor_id")
		
		if db.DB.Dialector.Name() == "postgres" {
			// PostgreSQL Full Text Search across product name, desc, and vendor store name
			query = query.Where("to_tsvector('english', products.name || ' ' || coalesce(products.description, '') || ' ' || vendor_profiles.store_name) @@ websearch_to_tsquery('english', ?)", *filter.Search)
		} else {
			// SQLite/Generic fallback
			query = query.Where("(products.name LIKE ? OR products.description LIKE ? OR vendor_profiles.store_name LIKE ?)", searchTerm, searchTerm, searchTerm)
		}
	}

	// Distinct by vendor if requested (Task 3 & 4)
	if filter.UniqueVendors {
		if db.DB.Dialector.Name() == "postgres" {
			// Postgres DISTINCT ON
			query = query.Distinct("ON (products.vendor_id) products.*").Order("products.vendor_id, products.created_at DESC")
		} else {
			// SQLite/Generic grouping (less precise but works)
			query = query.Group("products.vendor_id")
		}
	}

	// Count Total
	if err := query.Count(&totalItems).Error; err != nil {
		return nil, err
	}

	// Pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	// Execute
	if err := query.Order("created_at desc").Limit(limit).Offset(offset).Preload("Vendor").Find(&products).Error; err != nil {
		return nil, err
	}

	// Image fallback & Aliases
	for i := range products {
		if products[i].ImageURL == "" {
			products[i].ImageURL = s.config.DefaultProductImage
		}
		// Populate aliases for frontend compatibility
		products[i].Stock = products[i].StockQuantity
		products[i].MOQ = products[i].MinimumOrderQuantity
	}

	return &PaginatedResponse{
		Data: products,
		Meta: PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  int(totalItems),
			Limit:       limit,
		},
	}, nil
}

func (s *ProductService) GetProduct(id uuid.UUID) (*models.Product, error) {
	key := "product:" + id.String()
	ctx := context.Background()

	// 1. Try Redis
	if s.redis != nil {
		val, err := s.redis.Get(ctx, key).Result()
		if err == nil {
			var product models.Product
			if err := json.Unmarshal([]byte(val), &product); err == nil {
				return &product, nil
			}
		}
	}

	// 2. Fetch from DB
	var product models.Product
	if err := db.DB.Preload("Vendor").First(&product, id).Error; err != nil {
		return nil, errors.New("product not found")
	}

	// Populate aliases
	product.Stock = product.StockQuantity
	product.MOQ = product.MinimumOrderQuantity

	// 3. Cache in Redis
	if s.redis != nil {
		if data, err := json.Marshal(product); err == nil {
			s.redis.Set(ctx, key, data, time.Hour)
		}
	}

	return &product, nil
}

func (s *ProductService) ListVendors(filter VendorFilter) (*PaginatedResponse, error) {
	var vendors []models.VendorProfile
	var totalItems int64

	query := db.DB.Model(&models.VendorProfile{}).Where("is_approved = ?", true)

	if filter.BusinessType != nil && *filter.BusinessType != "" {
		query = query.Where("business_type = ?", *filter.BusinessType)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("store_name LIKE ?", search)
	}

	// Count
	if err := query.Count(&totalItems).Error; err != nil {
		return nil, err
	}

	// Pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	if err := query.Limit(limit).Offset(offset).Find(&vendors).Error; err != nil {
		return nil, err
	}

	return &PaginatedResponse{
		Data: vendors,
		Meta: PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  int(totalItems),
			Limit:       limit,
		},
	}, nil
}

func (s *ProductService) GetVendor(id uuid.UUID) (*models.VendorProfile, error) {
	var vendor models.VendorProfile
	if err := db.DB.Preload("Products", "is_active = ?", true).First(&vendor, "user_id = ?", id).Error; err != nil {
		return nil, errors.New("vendor not found")
	}

	return &vendor, nil
}
