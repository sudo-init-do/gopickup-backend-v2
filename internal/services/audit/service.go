package audit

import (
	"encoding/json"
	"gopickup/internal/models"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditService handles creating audit logs
type AuditService struct {
	db        *gorm.DB
	logChan   chan models.AuditLog
	batchSize int
}

// NewAuditService creates a new AuditService
func NewAuditService(db *gorm.DB) *AuditService {
	s := &AuditService{
		db:        db,
		logChan:   make(chan models.AuditLog, 1000), // Buffered channel
		batchSize: 10,
	}
	go s.processLogs()
	return s
}

func (s *AuditService) processLogs() {
	var batch []models.AuditLog
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case logEntry := <-s.logChan:
			batch = append(batch, logEntry)
			if len(batch) >= s.batchSize {
				s.flush(batch)
				batch = nil
			}
		case <-timer.C:
			if len(batch) > 0 {
				s.flush(batch)
				batch = nil
			}
		}
	}
}

func (s *AuditService) flush(logs []models.AuditLog) {
	// Simple retry mechanism
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := s.db.Create(&logs).Error; err != nil {
			log.Printf("Failed to flush audit logs (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Exponential backoff
			continue
		}
		return
	}
	log.Printf("Dropped %d audit logs after max retries", len(logs))
}

// Log records an action in the audit log asynchronously
func (s *AuditService) Log(actorID uuid.UUID, action, entityType string, entityID uuid.UUID, metadata interface{}) {
	var metaJSON []byte
	var err error

	if metadata != nil {
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			log.Printf("Failed to marshal audit metadata: %v", err)
			metaJSON = []byte("{}")
		}
	}

	auditLog := models.AuditLog{
		ActorUserID: actorID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Metadata:    metaJSON,
		// Timestamp should be set by GORM or here
		// CreatedAt: time.Now(), 
	}

	// Non-blocking send (if buffer full, log error and drop to avoid blocking main thread)
	select {
	case s.logChan <- auditLog:
	default:
		log.Printf("Audit log buffer full, dropping log: %s %s", action, entityType)
	}
}
