package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
)

type OutboxPublisher struct {
	db *gorm.DB
}

func NewOutboxPublisher(db *gorm.DB) *OutboxPublisher {
	return &OutboxPublisher{db: db}
}

// Publish writes the domain event to the outbox table within the current transaction.
func (p *OutboxPublisher) Publish(ctx context.Context, evt event.DomainEvent) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	outbox := OutboxModel{
		ID:        uuid.New(),
		EventType: evt.EventType(),
		Payload:   string(payload),
		Published: false,
		CreatedAt: time.Now(),
	}

	// Use transaction from context if available (standard GORM practice with DDD)
	db := p.db
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		db = tx
	}

	return db.WithContext(ctx).Create(&outbox).Error
}
