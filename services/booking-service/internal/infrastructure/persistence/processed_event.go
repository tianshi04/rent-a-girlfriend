package persistence

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ProcessedEventModel is a GORM model to store processed event IDs for idempotency check.
type ProcessedEventModel struct {
	EventID     string    `gorm:"column:event_id;type:varchar(100);primaryKey"`
	EventType   string    `gorm:"column:event_type;type:varchar(100);not null"`
	ProcessedAt time.Time `gorm:"column:processed_at;type:timestamptz;not null;autoCreateTime"`
}

func (ProcessedEventModel) TableName() string {
	return "processed_events"
}

// CheckAndRecordEvent check if event was already processed. If not, records it in the transaction.
// Returns (alreadyProcessed, error).
func CheckAndRecordEvent(ctx context.Context, db *gorm.DB, eventID string, eventType string) (bool, error) {
	tx := db.WithContext(ctx)
	if ctxTx, ok := ctx.Value("tx").(*gorm.DB); ok {
		tx = ctxTx.WithContext(ctx)
	}

	var count int64
	if err := tx.Model(&ProcessedEventModel{}).Where("event_id = ?", eventID).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil // Already processed
	}

	// Insert event_id
	record := ProcessedEventModel{
		EventID:     eventID,
		EventType:   eventType,
		ProcessedAt: time.Now(),
	}
	if err := tx.Create(&record).Error; err != nil {
		return false, err
	}
	return false, nil
}
