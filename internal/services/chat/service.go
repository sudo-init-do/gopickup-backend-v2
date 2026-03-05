package chat

import (
	"errors"
	"gopickup/internal/models"
	"gopickup/internal/services/notification"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatService struct {
	db           *gorm.DB
	notifService *notification.NotificationService
}

func NewChatService(db *gorm.DB, ns *notification.NotificationService) *ChatService {
	return &ChatService{
		db:           db,
		notifService: ns,
	}
}

// InitiateChat creates or returns an existing chat between two users, optionally linked to an order
func (s *ChatService) InitiateChat(initiatorID, recipientID uuid.UUID, orderID *uuid.UUID) (*models.Chat, error) {
	// 1. Validate users exist
	var initiator, recipient models.User
	if err := s.db.First(&initiator, "id = ?", initiatorID).Error; err != nil {
		return nil, errors.New("initiator not found")
	}
	if err := s.db.First(&recipient, "id = ?", recipientID).Error; err != nil {
		return nil, errors.New("recipient not found")
	}

	if initiatorID == recipientID {
		return nil, errors.New("cannot initiate chat with yourself")
	}

	// 2. Validate Order authorization if provided
	if orderID != nil {
		var order models.Order
		if err := s.db.First(&order, "id = ?", *orderID).Error; err != nil {
			return nil, errors.New("order not found")
		}

		// Check initiator role/relation
		if !isOrderActor(&initiator, &order) {
			return nil, errors.New("initiator is not authorized for this order")
		}
		// Check recipient role/relation
		if !isOrderActor(&recipient, &order) {
			return nil, errors.New("recipient is not authorized for this order")
		}
	}

	// 3. Check for existing chat
	// We need to find a chat that has BOTH participants and the same OrderID (or lack thereof)
	var existingChat models.Chat
	
	// Query chats with the specific OrderID (or NULL)
	query := s.db.Model(&models.Chat{}).
		Preload("Participants").
		Joins("JOIN chat_participants cp1 ON cp1.chat_id = chats.id AND cp1.user_id = ?", initiatorID).
		Joins("JOIN chat_participants cp2 ON cp2.chat_id = chats.id AND cp2.user_id = ?", recipientID)

	if orderID != nil {
		query = query.Where("chats.order_id = ?", *orderID)
	} else {
		query = query.Where("chats.order_id IS NULL")
	}

	if err := query.First(&existingChat).Error; err == nil {
		return &existingChat, nil
	}

	// 4. Create new chat
	newChat := models.Chat{
		OrderID: orderID,
		Participants: []models.User{
			{ID: initiatorID},
			{ID: recipientID},
		},
	}

	if err := s.db.Create(&newChat).Error; err != nil {
		return nil, err
	}

	return &newChat, nil
}

// GetUserChats returns all chats for a user
func (s *ChatService) GetUserChats(userID uuid.UUID, page, limit int) ([]ChatResponse, error) {
	// 1. Get user role
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	var chats []models.Chat
	offset := (page - 1) * limit

	var err error
	if user.Role == models.RoleAdmin {
		// Admin sees all chats
		err = s.db.Model(&models.Chat{}).
			Preload("Participants").
			Preload("Messages", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at DESC").Limit(1)
			}).
			Order("updated_at DESC").
			Offset(offset).Limit(limit).
			Find(&chats).Error
	} else {
		// Regular user sees their chats
		err = s.db.Model(&models.Chat{}).
			Joins("JOIN chat_participants cp ON cp.chat_id = chats.id AND cp.user_id = ?", userID).
			Preload("Participants").
			Preload("Messages", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at DESC").Limit(1)
			}).
			Order("updated_at DESC").
			Offset(offset).Limit(limit).
			Find(&chats).Error
	}

	if err != nil {
		return nil, err
	}

	var response []ChatResponse
	for _, chat := range chats {
		// Identify other participant
		var otherParticipant *models.User
		
		if user.Role == models.RoleAdmin {
			// For admin, just pick the first participant (or maybe the initiator?)
			// Ideally we should show both, but sticking to the response format:
			if len(chat.Participants) > 0 {
				otherParticipant = &chat.Participants[0]
			}
		} else {
			for _, p := range chat.Participants {
				if p.ID != userID {
					otherParticipant = &p
					break // Just pick the first one that isn't me
				}
			}
		}
		
		// If no other participant found (weird case, maybe self chat allowed later or data issue), skip or handle
		if otherParticipant == nil && len(chat.Participants) > 0 {
			// Maybe it's a chat with deleted user? Or just pick the first one if logic failed
			// For 1-on-1 chats, this logic holds.
		}

		resp := ChatResponse{
			ID:        chat.ID,
			OrderID:   chat.OrderID,
			CreatedAt: chat.CreatedAt,
			UpdatedAt: chat.UpdatedAt,
		}

		if otherParticipant != nil {
			resp.OtherParticipant = &UserSummary{
				ID:    otherParticipant.ID,
				Email: otherParticipant.Email,
				Role:  string(otherParticipant.Role),
				// Add name/avatar if available (requires querying profiles, but let's keep it simple for now or join profiles)
			}
		}

		if len(chat.Messages) > 0 {
			lastMsg := chat.Messages[0]
			resp.LastMessage = &MessagePreview{
				Content:   lastMsg.Content,
				CreatedAt: lastMsg.CreatedAt,
				SenderID:  lastMsg.SenderID,
				IsRead:    lastMsg.IsRead,
			}
		}
		
		// Calculate unread count
		// For Admin, unread count might not make sense in the same way, or maybe sum of unread messages?
		// Let's keep it 0 for admin or count all unread messages in the chat?
		// Requirement says "unread count (optional)".
		// For regular user: messages not sent by me and not read.
		if user.Role != models.RoleAdmin {
			var unreadCount int64
			s.db.Model(&models.Message{}).
				Where("chat_id = ? AND is_read = ? AND sender_id != ?", chat.ID, false, userID).
				Count(&unreadCount)
			resp.UnreadCount = int(unreadCount)
		}

		response = append(response, resp)
	}

	return response, nil
}

// GetChatMessages returns messages for a chat
func (s *ChatService) GetChatMessages(chatID, userID uuid.UUID, page, limit int) ([]models.Message, error) {
	// 1. Get user role
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Verify participation (skip for admin)
	if user.Role != models.RoleAdmin {
		if !s.isParticipant(chatID, userID) {
			return nil, errors.New("unauthorized: not a participant")
		}
	}

	var messages []models.Message
	offset := (page - 1) * limit

	// Order by newest first
	err := s.db.Where("chat_id = ?", chatID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&messages).Error

	return messages, err
}

// MarkMessagesRead marks all messages in a chat as read for the user
func (s *ChatService) MarkMessagesRead(chatID, userID uuid.UUID) error {
	// Verify participation
	if !s.isParticipant(chatID, userID) {
		return errors.New("unauthorized: not a participant")
	}

	// Update messages sent by others
	return s.db.Model(&models.Message{}).
		Where("chat_id = ? AND sender_id != ? AND is_read = ?", chatID, userID, false).
		Update("is_read", true).Error
}

// Helper: Check if user is participant
func (s *ChatService) isParticipant(chatID, userID uuid.UUID) bool {
	var count int64
	s.db.Table("chat_participants").
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Count(&count)
	return count > 0
}

// Helper: Check if user is actor on order
func isOrderActor(user *models.User, order *models.Order) bool {
	// Admin check
	if user.Role == models.RoleAdmin {
		return true
	}
	
	if order.ClientID == user.ID {
		return true
	}
	if order.VendorID == user.ID {
		return true
	}
	if order.DriverID != nil && *order.DriverID == user.ID {
		return true
	}
	return false
}

// Response structs
type ChatResponse struct {
	ID               uuid.UUID       `json:"id"`
	OrderID          *uuid.UUID      `json:"order_id,omitempty"`
	OtherParticipant *UserSummary    `json:"other_participant,omitempty"`
	LastMessage      *MessagePreview `json:"last_message,omitempty"`
	UnreadCount      int             `json:"unread_count"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type UserSummary struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

type MessagePreview struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	SenderID  uuid.UUID `json:"sender_id"`
	IsRead    bool      `json:"is_read"`
}
