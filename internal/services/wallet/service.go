package wallet

import (
	"errors"
	"gopickup/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletService struct {
	db *gorm.DB
}

func NewWalletService(db *gorm.DB) *WalletService {
	return &WalletService{db: db}
}

// GetBalance returns the user's wallet balance.
// It creates a wallet if one does not exist.
func (s *WalletService) GetBalance(userID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	err := s.db.Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create wallet if not exists
			wallet = models.Wallet{
				UserID:  userID,
				Balance: 0.0,
			}
			if createErr := s.db.Create(&wallet).Error; createErr != nil {
				return nil, createErr
			}
			return &wallet, nil
		}
		return nil, err
	}
	return &wallet, nil
}

// GetTransactions returns the transaction history for a user's wallet.
func (s *WalletService) GetTransactions(userID uuid.UUID, page, limit int) ([]models.Transaction, int64, error) {
	var wallet models.Wallet
	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		// If wallet doesn't exist, return empty list instead of error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Transaction{}, 0, nil
		}
		return nil, 0, err
	}

	var total int64

	offset := (page - 1) * limit

	if err := s.db.Model(&models.Transaction{}).Where("wallet_id = ?", wallet.ID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	transactions := []models.Transaction{}
	if err := s.db.Where("wallet_id = ?", wallet.ID).
		Order("created_at desc").
		Offset(offset).
		Limit(limit).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
