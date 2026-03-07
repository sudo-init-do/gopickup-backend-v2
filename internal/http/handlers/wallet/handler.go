package wallet

import (
	"gopickup/internal/services/wallet"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WalletHandler struct {
	service *wallet.WalletService
}

func NewWalletHandler(service *wallet.WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

// GetBalance returns the user's wallet balance
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	wallet, err := h.service.GetBalance(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get wallet balance"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// GetTransactions returns the transaction history
func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	transactions, total, err := h.service.GetTransactions(userID.(uuid.UUID), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": transactions,
		"meta": gin.H{
			"current_page": page,
			"limit":        limit,
			"total_items":  total,
			"total_pages":  (total + int64(limit) - 1) / int64(limit),
		},
	})
}
