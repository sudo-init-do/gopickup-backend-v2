package audit

import (
	"encoding/json"
	"gopickup/internal/models"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditService handles creating audit logs
type AuditService struct {
	db *gorm.DB
}

// NewAuditService creates a new AuditService
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// Log records an action in the audit log asynchronously
func (s *AuditService) Log(actorID uuid.UUID, action, entityType string, entityID uuid.UUID, metadata interface{}) {
	// Async logging to avoid blocking main request flow
	go func() {
		var metaJSON []byte
		var err error

		if metadata != nil {
			metaJSON, err = json.Marshal(metadata)
			if err != nil {
				log.Printf("Failed to marshal audit metadata: %v", err)
				// Continue with empty metadata or handle error appropriately
				metaJSON = []byte("{}")
			}
		}

		auditLog := models.AuditLog{
			ActorUserID: actorID,
			Action:      action,
			EntityType:  entityType,
			EntityID:    entityID,
			Metadata:    metaJSON,
		}

		if err := s.db.Create(&auditLog).Error; err != nil {
			log.Printf("Failed to create audit log: %v", err)
		}
	}()
}
