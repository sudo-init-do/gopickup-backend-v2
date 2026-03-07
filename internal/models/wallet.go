package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Wallet struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Balance   float64   `gorm:"type:decimal(15,2);default:0.00" json:"balance"`
	Currency  string    `gorm:"type:varchar(3);default:'NGN'" json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TransactionType string

const (
	TransactionCredit TransactionType = "credit"
	TransactionDebit  TransactionType = "debit"
)

type Transaction struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	WalletID    uuid.UUID       `gorm:"type:uuid;index;not null" json:"wallet_id"`
	Amount      float64         `gorm:"type:decimal(15,2);not null" json:"amount"`
	Type        TransactionType `gorm:"type:varchar(10);not null" json:"type"`
	Description string          `gorm:"type:text" json:"description"`
	Reference   string          `gorm:"type:varchar(100);index" json:"reference"`
	Status      string          `gorm:"type:varchar(20);default:'success'" json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) (err error) {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}
