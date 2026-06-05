package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

type OutboxPublisher struct {
	db *gorm.DB
}

func NewOutboxPublisher(db *gorm.DB) *OutboxPublisher {
	return &OutboxPublisher{db: db}
}

// Publish writes the event to the outbox table within the current transaction.
func (p *OutboxPublisher) Publish(ctx context.Context, evt event.DomainEvent) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	outbox := OutboxModel{
		ID:            uuid.New().String(),
		AggregateType: "Booking",
		AggregateID:   extractBookingID(evt), // defined in booking_repo_impl.go
		EventType:     evt.EventType(),
		Payload:       string(payload),
		Published:     false,
		CreatedAt:     time.Now(),
	}

	// Use transaction from context if available
	db := p.db
	if tx, ok := ctx.Value(vo.TxKey).(*gorm.DB); ok {
		db = tx
	}

	return db.WithContext(ctx).Create(&outbox).Error
}
