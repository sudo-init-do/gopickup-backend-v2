package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Chat struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID   *uuid.UUID `gorm:"type:uuid;index"` // Optional: Link to an order
	
	// For simplicity in this phase, we'll rely on Participants table or logic.
	// But standard GORM many-to-many:
	Participants []User `gorm:"many2many:chat_participants;"`

	Messages []Message `gorm:"foreignKey:ChatID"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (c *Chat) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}

type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChatID    uuid.UUID `gorm:"type:uuid;not null;index"`
	SenderID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Content   string    `gorm:"type:text;not null"`
	
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
