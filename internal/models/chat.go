package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Chat struct {
	ID      uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID *uuid.UUID `gorm:"type:uuid;index" json:"order_id"` // Optional: Link to an order

	// For simplicity in this phase, we'll rely on Participants table or logic.
	// But standard GORM many-to-many:
	Participants []User `gorm:"many2many:chat_participants;" json:"participants,omitempty"`

	Messages []Message `gorm:"foreignKey:ChatID" json:"messages,omitempty"`

	CreatedAt time.Time  `gorm:"index" json:"created_at"` // Indexed for sorting
	UpdatedAt time.Time  `json:"updated_at"` 
	DeletedAt *time.Time `gorm:"index" json:"-"` 
}

func (c *Chat) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}

type Message struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ChatID   uuid.UUID `gorm:"type:uuid;not null;index" json:"chat_id"`
	SenderID uuid.UUID `gorm:"type:uuid;not null;index" json:"sender_id"`
	Content  string    `gorm:"type:text;not null" json:"content"`
	IsRead   bool      `gorm:"default:false;index" json:"is_read"` // Indexed for unread count

	CreatedAt time.Time  `gorm:"index" json:"created_at"` // Indexed for sorting
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
