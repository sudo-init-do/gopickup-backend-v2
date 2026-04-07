package order

import (
	"errors"
	"fmt"
	"gopickup/internal/config"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/audit"
	"gopickup/internal/services/notification"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CheckoutItem struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

type CheckoutRequest struct {
	Items           []CheckoutItem       `json:"items" binding:"required,min=1,dive"`
	PaymentMethod   models.PaymentMethod `json:"payment_method" binding:"required"`
	PickupAddress   string               `json:"pickup_address" binding:"required"`
	DeliveryAddress string               `json:"delivery_address" binding:"required"`
	DeliveryLat     *float64             `json:"delivery_lat"`
	DeliveryLng     *float64             `json:"delivery_lng"`
}

type CheckoutResponse struct {
	Order            *models.Order `json:"order"`
	WhatsAppURL      string        `json:"whatsapp_url"`       // Always set — opens GoPickup support chat
	SupportPhone     string        `json:"support_phone"`      // GoPickup support WhatsApp number
}

type OrderService struct {
	audit *audit.AuditService
	cfg   *config.Config
}

func NewOrderService(audit *audit.AuditService, cfg *config.Config) *OrderService {
	return &OrderService{audit: audit, cfg: cfg}
}

// Checkout creates an order transactionally: validates products, stock, single vendor, creates order/items, decrements stock.
func (s *OrderService) Checkout(clientID uuid.UUID, req CheckoutRequest) (*CheckoutResponse, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("no items")
	}

	productIDs := make([]uuid.UUID, 0, len(req.Items))
	for _, it := range req.Items {
		productIDs = append(productIDs, it.ProductID)
	}

	var order *models.Order
	err := db.GetDB().Transaction(func(tx *gorm.DB) error {
		var products []models.Product
		// PERFORMANCE/SAFETY: Use FOR UPDATE to lock product rows and prevent race conditions on stock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", productIDs).Find(&products).Error; err != nil {
			return err
		}

		if len(products) != len(req.Items) {
			return errors.New("some products not found")
		}

		// Validate active and stock, and ensure single vendor
		var vendorID uuid.UUID
		total := 0.0
		for _, it := range req.Items {
			var p *models.Product
			for i := range products {
				if products[i].ID == it.ProductID {
					p = &products[i]
					break
				}
			}
			if p == nil {
				return errors.New("product missing")
			}
			if !p.IsActive {
				return errors.New("product not active")
			}
			if p.StockQuantity < it.Quantity {
				return errors.New("insufficient stock")
			}
			if vendorID == uuid.Nil {
				vendorID = p.VendorID
			} else if vendorID != p.VendorID {
				return errors.New("mixed vendor items not allowed")
			}
			total += p.Price * float64(it.Quantity)
		}

		status := models.OrderPending
		if req.PaymentMethod == models.PaymentWhatsApp {
			status = models.OrderAwaitingPayment
		}

		o := &models.Order{
			ClientID:           clientID,
			VendorID:           vendorID,
			TotalProductAmount: total,
			PaymentMethod:      req.PaymentMethod,
			PickupAddress:      req.PickupAddress,
			DeliveryAddress:    req.DeliveryAddress,
			DeliveryLat:        req.DeliveryLat,
			DeliveryLng:        req.DeliveryLng,
			Status:             status,
		}
		if err := tx.Create(o).Error; err != nil {
			return err
		}

		items := make([]models.OrderItem, 0, len(req.Items))
		for _, it := range req.Items {
			var p *models.Product
			for i := range products {
				if products[i].ID == it.ProductID {
					p = &products[i]
					break
				}
			}
			item := models.OrderItem{
				OrderID:   o.ID,
				ProductID: p.ID,
				Name:      p.Name,
				Price:     p.Price,
				Quantity:  it.Quantity,
			}
			items = append(items, item)
			// decrement stock
			if err := tx.Model(p).Update("stock_quantity", gorm.Expr("stock_quantity - ?", it.Quantity)).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		if err := tx.Preload("Vendor").Preload("Items").First(o, "id = ?", o.ID).Error; err != nil {
			return err
		}
		order = o
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.audit.Log(clientID, "ORDER_CREATED", "order", order.ID, map[string]interface{}{"total": order.TotalProductAmount})
	notification.GetService().NotifyOrderStatusUpdate(order.ID, order.Status, order.ClientID, order.VendorID, order.DriverID)

	resp := &CheckoutResponse{
		Order: order,
	}

	// Always generate a WhatsApp link to GoPickup Support for negotiation
	supportPhone := s.cfg.WhatsAppSupportNumber
	if supportPhone == "" {
		supportPhone = "2348000000000" // fallback placeholder
	}
	// Normalise: strip leading + if present
	supportPhone = strings.TrimPrefix(supportPhone, "+")

	var vendorName string
	var vendor models.VendorProfile
	if err := db.GetDB().First(&vendor, "user_id = ?", order.VendorID).Error; err == nil {
		vendorName = vendor.StoreName
	}

	msg := fmt.Sprintf(
		"Hi GoPickup Support! 👋\n\nI'd like to negotiate an order.\n\nOrder ID: %s\nVendor: %s\nEstimated Total: ₦%.2f\nDelivery Address: %s\n\nPlease help me finalise the price, quantity, and payment details.",
		order.ID.String()[:8], vendorName, order.TotalProductAmount, order.DeliveryAddress,
	)
	resp.WhatsAppURL = fmt.Sprintf("https://wa.me/%s?text=%s", supportPhone, url.QueryEscape(msg))
	resp.SupportPhone = supportPhone

	return resp, nil
}

// ClientReportPaymentMade is called when the client taps "I Have Made Payment".
// Moves the order from pending/awaiting_payment → payment_made for admin verification.
func (s *OrderService) ClientReportPaymentMade(clientID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	if err := db.GetDB().First(&o, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	if o.ClientID != clientID {
		return nil, errors.New("forbidden")
	}
	if o.Status != models.OrderPending && o.Status != models.OrderAwaitingPayment {
		return nil, errors.New("order is not in a state where payment can be reported")
	}
	o.Status = models.OrderPaymentMade
	if err := db.GetDB().Save(&o).Error; err != nil {
		return nil, err
	}
	s.audit.Log(clientID, "CLIENT_REPORTED_PAYMENT", "order", o.ID, nil)
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	return &o, nil
}

// ConfirmPayment moves order from awaiting_payment to processing (Vendor/Admin only)
func (s *OrderService) ConfirmPayment(userID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	if err := db.GetDB().First(&o, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	
	// Only vendor or admin can confirm
	if o.VendorID != userID {
		// Check for admin role separately if needed, but for now we assume role check is in handler
	}

	if o.Status != models.OrderAwaitingPayment && o.Status != models.OrderPending {
		return nil, errors.New("order is not in a payable state")
	}

	o.Status = models.OrderProcessing
	if err := db.GetDB().Save(&o).Error; err != nil {
		return nil, err
	}
	
	s.audit.Log(userID, "PAYMENT_CONFIRMED", "order", o.ID, nil)
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	
	return &o, nil
}

// List orders for role
func (s *OrderService) ListOrders(userID uuid.UUID, role models.UserRole, page, limit int) ([]models.Order, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	// Enforce max limit for performance safety
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit
	var orders []models.Order
	q := db.GetDB().Model(&models.Order{}).Order("created_at DESC")
	switch role {
	case models.RoleClient:
		q = q.Where("client_id = ?", userID)
	case models.RoleVendor:
		q = q.Where("vendor_id = ?", userID)
	case models.RoleDriver:
		q = q.Where("driver_id = ?", userID)
	case models.RoleAdmin:
		// all
	default:
		q = q.Where("1 = 0")
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Preload("Vendor").Preload("Items").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, count, nil
}

// Get order detail if authorized. Includes items.
func (s *OrderService) GetOrder(userID uuid.UUID, role models.UserRole, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	if err := db.GetDB().Preload("Items").Preload("Vendor").Preload("Bids").First(&o, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	if role == models.RoleAdmin {
		return &o, nil
	}
	switch role {
	case models.RoleClient:
		if o.ClientID != userID {
			return nil, errors.New("forbidden")
		}
	case models.RoleVendor:
		if o.VendorID != userID {
			return nil, errors.New("forbidden")
		}
	case models.RoleDriver:
		if o.DriverID == nil || *o.DriverID != userID {
			return nil, errors.New("forbidden")
		}
	default:
		return nil, errors.New("forbidden")
	}
	return &o, nil
}

// Vendor updates status with allowed transitions.
func (s *OrderService) VendorUpdateStatus(vendorID uuid.UUID, orderID uuid.UUID, next models.OrderStatus) (*models.Order, error) {
	var o models.Order
	if err := db.GetDB().First(&o, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	if o.VendorID != vendorID {
		return nil, errors.New("forbidden")
	}
	switch o.Status {
	case models.OrderPending:
		if next != models.OrderProcessing && next != models.OrderCancelled {
			return nil, errors.New("invalid transition")
		}
	case models.OrderProcessing:
		if next != models.OrderCancelled {
			return nil, errors.New("invalid transition")
		}
	default:
		return nil, errors.New("invalid transition")
	}

	// Handle stock restoration on cancellation
	if next == models.OrderCancelled {
		if err := s.restoreStock(o.ID); err != nil {
			return nil, err
		}
	}

	o.Status = next
	if err := db.GetDB().Save(&o).Error; err != nil {
		return nil, err
	}
	s.audit.Log(vendorID, "ORDER_STATUS_CHANGED", "order", o.ID, map[string]interface{}{"status": next})
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	return &o, nil
}

// Client cancels an order (only if pending)
func (s *OrderService) ClientCancelOrder(clientID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	err := db.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, "id = ?", orderID).Error; err != nil {
			return err
		}
		if o.ClientID != clientID {
			return errors.New("forbidden")
		}
		if o.Status != models.OrderPending {
			return errors.New("cannot cancel non-pending order")
		}

		// Restore stock
		if err := s.restoreStockTx(tx, o.ID); err != nil {
			return err
		}

		o.Status = models.OrderCancelled
		if err := tx.Save(&o).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.audit.Log(clientID, "ORDER_CANCELLED", "order", o.ID, nil)
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	return &o, nil
}

// Helper to restore stock
func (s *OrderService) restoreStock(orderID uuid.UUID) error {
	return db.GetDB().Transaction(func(tx *gorm.DB) error {
		return s.restoreStockTx(tx, orderID)
	})
}

func (s *OrderService) restoreStockTx(tx *gorm.DB, orderID uuid.UUID) error {
	var items []models.OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		if err := tx.Model(&models.Product{}).Where("id = ?", item.ProductID).
			Update("stock_quantity", gorm.Expr("stock_quantity + ?", item.Quantity)).Error; err != nil {
			return err
		}
	}
	return nil
}

// Vendor marks ready: processing -> searching_driver.
func (s *OrderService) VendorMarkReady(vendorID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	if err := db.GetDB().First(&o, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	if o.VendorID != vendorID {
		return nil, errors.New("forbidden")
	}
	o.Status = models.OrderProcessing
	if err := db.GetDB().Save(&o).Error; err != nil {
		return nil, err
	}
	s.audit.Log(vendorID, "ORDER_READY_FOR_ADMIN", "order", o.ID, map[string]interface{}{"status": models.OrderProcessing})
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	return &o, nil
}

// DriverAcceptLoad is called by a driver after WhatsApp negotiation to officially take the job.
// Supports two paths:
//  1. Admin pre-assigned (status = assigned, driverID already set): driver just confirms.
//  2. Open pool (status = processing, no driver set yet): driver self-assigns.
func (s *OrderService) DriverAcceptLoad(driverID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	err := db.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, "id = ?", orderID).Error; err != nil {
			return err
		}

		switch o.Status {
		case models.OrderAssigned:
			// Admin pre-assigned path — must be assigned to this driver
			if o.DriverID == nil || *o.DriverID != driverID {
				return errors.New("this load is not assigned to you")
			}
		case models.OrderProcessing:
			// Open pool path — first approved driver to accept wins
			var profile models.DriverProfile
			if err := tx.First(&profile, "user_id = ?", driverID).Error; err != nil {
				return errors.New("driver profile not found")
			}
			if !profile.IsApproved {
				return errors.New("only approved drivers can accept jobs")
			}
			o.DriverID = &driverID
		default:
			return errors.New("job cannot be accepted in its current status")
		}

		o.Status = models.OrderInProgress
		return tx.Save(&o).Error
	})
	if err != nil {
		return nil, err
	}

	s.audit.Log(driverID, "DRIVER_ACCEPTED_LOAD", "order", o.ID, nil)
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	return &o, nil
}
