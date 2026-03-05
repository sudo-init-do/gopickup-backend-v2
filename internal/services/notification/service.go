package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gopickup/internal/models"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

type NotificationService struct {
	hub *Hub
	db  *gorm.DB
	fcm *messaging.Client
}

var instance *NotificationService

func NewNotificationService(database *gorm.DB) *NotificationService {
	hub := NewHub()
	go hub.Run()

	var fcmClient *messaging.Client

	// Initialize FCM if credentials exist
	// In a real app, we'd load credentials from a file or env
	// For now, we'll just log that FCM is not configured if init fails
	// Or we can try to init with default credentials (GOOGLE_APPLICATION_CREDENTIALS)
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile("firebase-credentials.json"))
	if err == nil {
		fcmClient, err = app.Messaging(ctx)
		if err != nil {
			log.Printf("Error initializing FCM messaging: %v", err)
		} else {
			log.Println("FCM initialized successfully")
		}
	} else {
		log.Printf("FCM not initialized (credentials missing?): %v", err)
	}

	instance = &NotificationService{
		hub: hub,
		db:  database,
		fcm: fcmClient,
	}
	return instance
}

func GetService() *NotificationService {
	return instance
}

func (s *NotificationService) GetHub() *Hub {
	return s.hub
}

// UpdateFCMToken updates the user's FCM token
func (s *NotificationService) UpdateFCMToken(userID uuid.UUID, token string) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", token).Error
}

// BroadcastToRoom sends a WebSocket message to a room
func (s *NotificationService) BroadcastToRoom(room string, event string, payload interface{}) {
	msg := map[string]interface{}{
		"event":   event,
		"payload": payload,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}
	s.hub.BroadcastToRoom(room, bytes)
}

// SendNotification sends an FCM notification and optional WS message
func (s *NotificationService) SendNotification(userID uuid.UUID, title, body string, data map[string]string) {
	// 1. Send WebSocket message (to user's private room)
	wsPayload := map[string]interface{}{
		"title": title,
		"body":  body,
		"data":  data,
	}
	s.BroadcastToRoom(fmt.Sprintf("user:%s", userID.String()), "notification", wsPayload)

	// 2. Send FCM
	if s.fcm == nil {
		// log.Println("FCM not configured, skipping push")
		return
	}

	var user models.User
	if err := s.db.Select("fcm_token").First(&user, "id = ?", userID).Error; err != nil {
		log.Printf("Error fetching user for notification: %v", err)
		return
	}

	if user.FCMToken == nil || *user.FCMToken == "" {
		log.Printf("User %s has no FCM token, skipping push", userID)
		return
	}

	msg := &messaging.Message{
		Token: *user.FCMToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	_, err := s.fcm.Send(context.Background(), msg)
	if err != nil {
		log.Printf("Error sending FCM to user %s: %v", userID, err)
	}
}

// Specific event helpers

func (s *NotificationService) NotifyOrderStatusUpdate(orderID uuid.UUID, status models.OrderStatus, clientID, vendorID uuid.UUID, driverID *uuid.UUID) {
	payload := map[string]interface{}{
		"order_id": orderID,
		"status":   status,
	}

	// Broadcast to order room
	s.BroadcastToRoom(fmt.Sprintf("order:%s", orderID.String()), "order_status_updated", payload)

	// Send Push Notifications
	// To Client
	s.SendNotification(clientID, "Order Update", fmt.Sprintf("Your order is now %s", status), map[string]string{
		"type": "order_status",
		"id":   orderID.String(),
	})

	// To Vendor? Maybe not needed as push, but WS is good.

	// To Driver if assigned
	if driverID != nil {
		s.SendNotification(*driverID, "Job Update", fmt.Sprintf("Order %s is now %s", orderID.String(), status), map[string]string{
			"type": "order_status",
			"id":   orderID.String(),
		})
	}
}

func (s *NotificationService) NotifyNewBid(clientID uuid.UUID, orderID uuid.UUID, amount float64) {
	s.SendNotification(clientID, "New Bid", fmt.Sprintf("A driver bid %.2f on your order", amount), map[string]string{
		"type":     "new_bid",
		"order_id": orderID.String(),
	})
	// Also WS
	s.BroadcastToRoom(fmt.Sprintf("user:%s", clientID.String()), "new_bid", map[string]interface{}{
		"order_id": orderID,
		"amount":   amount,
	})
}

func (s *NotificationService) NotifyBidAccepted(driverID uuid.UUID, orderID uuid.UUID) {
	s.SendNotification(driverID, "Bid Accepted", "Your bid has been accepted!", map[string]string{
		"type":     "bid_accepted",
		"order_id": orderID.String(),
	})
	// Also WS
	s.BroadcastToRoom(fmt.Sprintf("user:%s", driverID.String()), "bid_accepted", map[string]interface{}{
		"order_id": orderID,
	})
}

func (s *NotificationService) NotifyDriverMoved(orderID uuid.UUID, lat, lng float64) {
	// Only WS, no push
	s.BroadcastToRoom(fmt.Sprintf("order:%s", orderID.String()), "driver_moved", map[string]interface{}{
		"order_id": orderID,
		"lat":      lat,
		"lng":      lng,
		"ts":       time.Now(),
	})
}

func (s *NotificationService) NotifyNewMessage(chatID uuid.UUID, senderID uuid.UUID, content string, recipientIDs []uuid.UUID) {
	payload := map[string]interface{}{
		"chat_id":   chatID,
		"sender_id": senderID,
		"content":   content,
		"ts":        time.Now(),
	}

	// 1. Emit to chat room
	s.BroadcastToRoom(fmt.Sprintf("chat:%s", chatID.String()), "new_message", payload)

	// 2. Emit to recipient user rooms and send Push
	for _, rid := range recipientIDs {
		// Skip sender
		if rid == senderID {
			continue
		}

		// WS to user room (for UI triggers/badges)
		s.BroadcastToRoom(fmt.Sprintf("user:%s", rid.String()), "new_message", payload)

		// FCM
		s.SendNotification(rid, "New Message", "You have a new message", map[string]string{
			"type":    "chat_message",
			"chat_id": chatID.String(),
		})
	}
}
