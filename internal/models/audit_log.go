package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	ActorUserID uuid.UUID       `gorm:"type:uuid;index" json:"actor_user_id"`
	Action      string          `gorm:"type:varchar(255);index" json:"action"`
	EntityType  string          `gorm:"type:varchar(50);index" json:"entity_type"`
	EntityID    uuid.UUID       `gorm:"type:uuid;index" json:"entity_id"`
	Metadata    json.RawMessage `gorm:"type:jsonb" json:"metadata"` // Using json.RawMessage for flexible JSON storage
	CreatedAt   time.Time       `json:"created_at"`
}

// BeforeCreate ensures the AuditLog has a valid UUID
func (a *AuditLog) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}
