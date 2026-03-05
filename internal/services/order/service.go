package order

import (
	"errors"
	"gopickup/internal/db"
	"gopickup/internal/models"
	"gopickup/internal/services/notification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CheckoutItem struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

type CheckoutRequest struct {
	Items           []CheckoutItem     `json:"items" binding:"required,min=1,dive"`
	PaymentMethod   models.PaymentMethod `json:"payment_method" binding:"required"`
	PickupAddress   string             `json:"pickup_address" binding:"required"`
	DeliveryAddress string             `json:"delivery_address" binding:"required"`
	DeliveryLat     *float64           `json:"delivery_lat"`
	DeliveryLng     *float64           `json:"delivery_lng"`
}

type OrderService struct{}

func NewOrderService() *OrderService {
	return &OrderService{}
}

// Checkout creates an order transactionally: validates products, stock, single vendor, creates order/items, decrements stock.
func (s *OrderService) Checkout(clientID uuid.UUID, req CheckoutRequest) (*models.Order, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("no items")
	}

	var products []models.Product
	productIDs := make([]uuid.UUID, 0, len(req.Items))
	for _, it := range req.Items {
		productIDs = append(productIDs, it.ProductID)
	}

	err := db.GetDB().Where("id IN ?", productIDs).Find(&products).Error
	if err != nil {
		return nil, err
	}
	if len(products) != len(req.Items) {
		return nil, errors.New("some products not found")
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
			return nil, errors.New("product missing")
		}
		if !p.IsActive {
			return nil, errors.New("product not active")
		}
		if p.StockQuantity < it.Quantity {
			return nil, errors.New("insufficient stock")
		}
		if vendorID == uuid.Nil {
			vendorID = p.VendorID
		} else if vendorID != p.VendorID {
			return nil, errors.New("mixed vendor items not allowed")
		}
		total += p.Price * float64(it.Quantity)
	}

	var order *models.Order
	err = db.GetDB().Transaction(func(tx *gorm.DB) error {
		o := &models.Order{
			ClientID:           clientID,
			VendorID:           vendorID,
			TotalProductAmount: total,
			PaymentMethod:      req.PaymentMethod,
			PickupAddress:      req.PickupAddress,
			DeliveryAddress:    req.DeliveryAddress,
			DeliveryLat:        req.DeliveryLat,
			DeliveryLng:        req.DeliveryLng,
			Status:             models.OrderPending,
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
		order = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// List orders for role
func (s *OrderService) ListOrders(userID uuid.UUID, role models.UserRole, page, limit int) ([]models.Order, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
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
	if err := q.Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, count, nil
}

// Get order detail if authorized. Includes items.
func (s *OrderService) GetOrder(userID uuid.UUID, role models.UserRole, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	if err := db.GetDB().Preload("Items").First(&o, "id = ?", orderID).Error; err != nil {
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
	o.Status = next
	if err := db.GetDB().Save(&o).Error; err != nil {
		return nil, err
	}
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	return &o, nil
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
	if o.Status != models.OrderProcessing {
		return nil, errors.New("invalid transition")
	}
	o.Status = models.OrderSearchingDriver
	if err := db.GetDB().Save(&o).Error; err != nil {
		return nil, err
	}
	notification.GetService().NotifyOrderStatusUpdate(o.ID, o.Status, o.ClientID, o.VendorID, o.DriverID)
	return &o, nil
}

// GetBids returns all bids for a given order, ensuring the client owns it.
func (s *OrderService) GetBids(clientID uuid.UUID, orderID uuid.UUID) ([]models.Bid, error) {
	var o models.Order
	if err := db.GetDB().First(&o, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	if o.ClientID != clientID {
		return nil, errors.New("forbidden")
	}

	var bids []models.Bid
	if err := db.GetDB().Where("order_id = ?", orderID).Find(&bids).Error; err != nil {
		return nil, err
	}
	return bids, nil
}

// AcceptBid assigns a driver to an order based on a bid.
// It uses a transaction and optimistic locking (status check) to prevent race conditions.
func (s *OrderService) AcceptBid(clientID uuid.UUID, orderID uuid.UUID, bidID uuid.UUID) (*models.Order, error) {
	var order models.Order

	err := db.GetDB().Transaction(func(tx *gorm.DB) error {
		// 1. Fetch order to verify ownership
		if err := tx.First(&order, "id = ?", orderID).Error; err != nil {
			return err
		}
		if order.ClientID != clientID {
			return errors.New("forbidden")
		}

		// 2. Fetch bid
		var bid models.Bid
		if err := tx.First(&bid, "id = ?", bidID).Error; err != nil {
			return err
		}
		if bid.OrderID != orderID {
			return errors.New("bid does not belong to this order")
		}

		// 3. Update Order atomically (Race Condition Safety)
		// We use Where("status = ?", OrderSearchingDriver) to ensure it hasn't been taken by another request.
		res := tx.Model(&models.Order{}).
			Where("id = ? AND status = ?", orderID, models.OrderSearchingDriver).
			Updates(map[string]interface{}{
				"status":    models.OrderAssigned,
				"driver_id": bid.DriverID,
			})

		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("order is no longer available for assignment")
		}

		// 4. Mark Bid as Accepted
		if err := tx.Model(&bid).Update("status", models.BidAccepted).Error; err != nil {
			return err
		}

		// 5. Reject all other bids for this order
		if err := tx.Model(&models.Bid{}).
			Where("order_id = ? AND id != ?", orderID, bidID).
			Update("status", models.BidRejected).Error; err != nil {
			return err
		}

		// Refresh order object with new status/driver
		return tx.First(&order, "id = ?", orderID).Error
	})

	if err != nil {
		return nil, err
	}

	// Notifications
	ns := notification.GetService()
	if order.DriverID != nil {
		ns.NotifyBidAccepted(*order.DriverID, order.ID)
	}
	ns.NotifyOrderStatusUpdate(order.ID, order.Status, order.ClientID, order.VendorID, order.DriverID)

	return &order, nil
}
