package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gopickup/internal/config"
	"gopickup/internal/models"
	"gopickup/internal/services/driver"
	"gopickup/internal/services/notification"
	"gopickup/internal/services/order"
	"gopickup/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type Handler struct {
	notifService  *notification.NotificationService
	orderService  *order.OrderService
	driverService *driver.DriverService
	db            *gorm.DB
	cfg           *config.Config
}

func NewHandler(ns *notification.NotificationService, os *order.OrderService, ds *driver.DriverService, db *gorm.DB, cfg *config.Config) *Handler {
	return &Handler{
		notifService:  ns,
		orderService:  os,
		driverService: ds,
		db:            db,
		cfg:           cfg,
	}
}

// HandleConnection upgrades HTTP to WS
func (h *Handler) HandleConnection(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	claims, err := utils.ValidateJWT(token, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &notification.Client{
		Hub:    h.notifService.GetHub(),
		Send:   make(chan []byte, 256),
		UserID: claims.UserID.String(),
		Rooms:  make(map[string]bool),
	}

	client.Hub.Register <- client

	// Auto-join user room
	h.notifService.GetHub().Subscribe(client, fmt.Sprintf("user:%s", client.UserID))

	// Start goroutines for reading and writing
	go h.writePump(client, conn)
	go h.readPump(client, conn, claims.UserID)
}

func (h *Handler) readPump(client *notification.Client, conn *websocket.Conn, userID uuid.UUID) {
	defer func() {
		client.Hub.Unregister <- client
		conn.Close()
	}()

	conn.SetReadLimit(5120) // 5KB limit
	if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("SetReadDeadline failed: %v", err)
		return
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("SetReadDeadline failed in PongHandler: %v", err)
			return err
		}
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Handle client -> server messages
		h.handleMessage(client, userID, message)
	}
}

func (h *Handler) writePump(client *notification.Client, conn *websocket.Conn) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			// Add queued chat messages to the current websocket message.
			// n := len(client.Send)
			// for i := 0; i < n; i++ {
			// 	w.Write([]byte{'\n'})
			// 	w.Write(<-client.Send)
			// }

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type WSEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func (h *Handler) handleMessage(client *notification.Client, userID uuid.UUID, message []byte) {
	var event WSEvent
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid JSON: %v", err)
		return
	}

	switch event.Event {
	case "join_order_room":
		h.handleJoinOrderRoom(client, userID, event.Payload)
	case "leave_order_room":
		h.handleLeaveOrderRoom(client, userID, event.Payload)
	case "join_chat_room":
		h.handleJoinChatRoom(client, userID, event.Payload)
	case "leave_chat_room":
		h.handleLeaveChatRoom(client, userID, event.Payload)
	case "driver_location_update":
		h.handleDriverLocationUpdate(client, userID, event.Payload)
	case "chat_message":
		h.handleChatMessage(client, userID, event.Payload)
	default:
		log.Printf("Unknown event: %s", event.Event)
	}
}

// Event Handlers implementation

func (h *Handler) handleJoinOrderRoom(client *notification.Client, userID uuid.UUID, payload json.RawMessage) {
	var p struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}

	// Authorization check: Is user related to order?
	// Client, Vendor, Driver, Admin
	// We'll skip Admin check for now or assume Admin role if needed (but we don't have role here easily unless we store it in Client)
	// Let's check DB
	var order models.Order
	if err := h.db.First(&order, "id = ?", p.OrderID).Error; err != nil {
		return
	}

	isAuthorized := false
	if order.ClientID == userID || order.VendorID == userID {
		isAuthorized = true
	} else if order.DriverID != nil && *order.DriverID == userID {
		isAuthorized = true
	}

	if isAuthorized {
		h.notifService.GetHub().Subscribe(client, fmt.Sprintf("order:%s", p.OrderID))
	}
}

func (h *Handler) handleLeaveOrderRoom(client *notification.Client, userID uuid.UUID, payload json.RawMessage) {
	var p struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	h.notifService.GetHub().Unsubscribe(client, fmt.Sprintf("order:%s", p.OrderID))
}

func (h *Handler) handleJoinChatRoom(client *notification.Client, userID uuid.UUID, payload json.RawMessage) {
	var p struct {
		ChatID string `json:"chat_id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}

	// Auth check: Is user participant?
	// Assuming Chat model has participants logic or simple check
	// Since we defined Chat model but no participants table yet (just commented),
	// let's assume if we had a Participants relation.
	// For now, let's allow join if Chat exists, but ideally check participants.
	// Or better: check if chat is linked to an order user is part of.
	var chat models.Chat
	if err := h.db.First(&chat, "id = ?", p.ChatID).Error; err != nil {
		return
	}

	// If Chat has OrderID, check order participants
	if chat.OrderID != nil {
		var order models.Order
		if err := h.db.First(&order, "id = ?", *chat.OrderID).Error; err == nil {
			if order.ClientID == userID || order.VendorID == userID || (order.DriverID != nil && *order.DriverID == userID) {
				log.Printf("User %s authorized for chat %s", userID, p.ChatID)
				h.notifService.GetHub().Subscribe(client, fmt.Sprintf("chat:%s", p.ChatID))
			} else {
				log.Printf("User %s NOT authorized for chat %s (ClientID=%s, VendorID=%s, DriverID=%v)", userID, p.ChatID, order.ClientID, order.VendorID, order.DriverID)
			}
		} else {
			log.Printf("Order %s not found for chat %s", *chat.OrderID, p.ChatID)
		}
		// Generic chat? Check participants manually if implemented.
		// For Phase 8 MVP, let's assume Order-based chats are the main use case.
		// Or allow join if user sent a message there before?
		// We'll skip strict check for generic chats for now or block them.
	}
}

func (h *Handler) handleLeaveChatRoom(client *notification.Client, userID uuid.UUID, payload json.RawMessage) {
	var p struct {
		ChatID string `json:"chat_id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	h.notifService.GetHub().Unsubscribe(client, fmt.Sprintf("chat:%s", p.ChatID))
}

func (h *Handler) handleDriverLocationUpdate(client *notification.Client, userID uuid.UUID, payload json.RawMessage) {
	var p struct {
		OrderID string  `json:"order_id"`
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}

	orderID, err := uuid.Parse(p.OrderID)
	if err != nil {
		return
	}

	// Check if user is the assigned driver and order is in transit
	var order models.Order
	if err := h.db.First(&order, "id = ?", orderID).Error; err != nil {
		return
	}

	if order.DriverID == nil || *order.DriverID != userID {
		// Not the assigned driver
		return
	}

	// "Only when order.status=transit"
	// Transit statuses: "picked_up", "delivered" (maybe too late?), "assigned" (maybe on way to pickup?)
	// Let's assume "assigned" (on way to pickup) and "picked_up" (on way to delivery) are valid.
	if order.Status != models.OrderAssigned && order.Status != models.OrderPickedUp {
		return
	}

	// Update driver location in DB
	if err := h.driverService.UpdateLocation(userID, p.Lat, p.Lng); err != nil {
		log.Printf("UpdateLocation failed: %v", err)
	}

	// Emit to order room
	h.notifService.NotifyDriverMoved(orderID, p.Lat, p.Lng)
}

func (h *Handler) handleChatMessage(client *notification.Client, userID uuid.UUID, payload json.RawMessage) {
	var p struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}

	chatID, err := uuid.Parse(p.ChatID)
	if err != nil {
		return
	}

	// Validate sender is participant
	var chat models.Chat
	if err := h.db.First(&chat, "id = ?", chatID).Error; err != nil {
		return
	}

	// Check authorization (similar to join)
	var recipientIDs []uuid.UUID

	if chat.OrderID != nil {
		var order models.Order
		if err := h.db.First(&order, "id = ?", *chat.OrderID).Error; err == nil {
			isParticipant := false

			// Add participants
			if order.ClientID == userID {
				isParticipant = true
			}
			recipientIDs = append(recipientIDs, order.ClientID)

			if order.VendorID == userID {
				isParticipant = true
			}
			recipientIDs = append(recipientIDs, order.VendorID)

			if order.DriverID != nil {
				if *order.DriverID == userID {
					isParticipant = true
				}
				recipientIDs = append(recipientIDs, *order.DriverID)
			}

			if !isParticipant {
				log.Printf("User %s is not a participant", userID)
				return // Not a participant
			}
		} else {
			log.Printf("Order not found: %v", err)
		}
	} else {
		// Generic check
		log.Println("Generic chat not supported yet")
		return
	}

	// Persist message
	msg := models.Message{
		ChatID:   chatID,
		SenderID: userID,
		Content:  p.Text,
	}
	if err := h.db.Create(&msg).Error; err != nil {
		log.Printf("Error saving message: %v", err)
		return
	}

	// Emit
	log.Printf("Emitting new message to chat %s and recipients %v", chatID, recipientIDs)
	h.notifService.NotifyNewMessage(chatID, userID, p.Text, recipientIDs)
}

func (h *Handler) UpdateFCMToken(c *gin.Context) {
	userIDInf, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDInf.(uuid.UUID)

	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.notifService.UpdateFCMToken(userID, req.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "FCM token updated"})
}
