package chat

import (
	"gopickup/internal/http/handlers"
	"gopickup/internal/models"
	"gopickup/internal/services/chat"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var _ = models.Chat{}

type Handler struct {
	service *chat.ChatService
}

func NewHandler(s *chat.ChatService) *Handler {
	return &Handler{service: s}
}

type InitiateChatRequest struct {
	RecipientID uuid.UUID  `json:"recipient_user_id" binding:"required"`
	OrderID     *uuid.UUID `json:"order_id"`
}

// InitiateChat godoc
// @Summary Initiate a chat
// @Description Start a new chat or get existing one
// @Tags chats
// @Accept json
// @Produce json
// @Param request body InitiateChatRequest true "Chat details"
// @Success 200 {object} chat.ChatResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Router /chats/initiate [post]
func (h *Handler) InitiateChat(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req InitiateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handlers.ErrorResponse{Error: err.Error()})
		return
	}

	chat, err := h.service.InitiateChat(userID, req.RecipientID, req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, handlers.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, chat)
}

// GetChats godoc
// @Summary Get user chats
// @Description List all chats for the current user
// @Tags chats
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {array} chat.ChatResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /chats [get]
func (h *Handler) GetChats(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	chats, err := h.service.GetUserChats(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chats)
}

// GetMessages godoc
// @Summary Get chat messages
// @Description Get history of a specific chat
// @Tags chats
// @Produce json
// @Param id path string true "Chat ID"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {array} models.Message
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /chats/{id}/messages [get]
func (h *Handler) GetMessages(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	chatID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	messages, err := h.service.GetChatMessages(chatID, userID, page, limit)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized: not a participant" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// MarkRead godoc
// @Summary Mark messages as read
// @Description Mark all messages in a chat as read
// @Tags chats
// @Produce json
// @Param id path string true "Chat ID"
// @Success 200 {object} handlers.SuccessResponse
// @Router /chats/{id}/read [patch]
func (h *Handler) MarkRead(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	chatID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id"})
		return
	}

	if err := h.service.MarkMessagesRead(chatID, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized: not a participant" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "messages marked as read"})
}
