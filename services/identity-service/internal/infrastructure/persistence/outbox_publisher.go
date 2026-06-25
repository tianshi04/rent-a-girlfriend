package persistence

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

type OutboxPublisher struct {
	db *gorm.DB
}

func NewOutboxPublisher(db *gorm.DB) *OutboxPublisher {
	return &OutboxPublisher{db: db}
}

// Publish writes the domain event to the outbox table within the current transaction.
func (p *OutboxPublisher) Publish(ctx context.Context, evt event.DomainEvent) error {
	var payload []byte
	var err error

	protoMsg := evt.ToProto()
	if protoMsg != nil && !reflect.ValueOf(protoMsg).IsNil() {
		payload, err = protojson.Marshal(protoMsg)
	} else {
		payload, err = json.Marshal(evt)
	}
	if err != nil {
		return err
	}

	correlationID := ""
	if val, ok := ctx.Value(vo.CorrelationIDKey).(string); ok {
		correlationID = val
	}

	outbox := OutboxModel{
		ID:            uuid.New(),
		EventType:     evt.EventType(),
		Payload:       string(payload),
		CorrelationID: correlationID,
		Published:     false,
		CreatedAt:     time.Now(),
	}

	// Use transaction from context if available (standard GORM practice with DDD)
	db := p.db
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		db = tx
	}

	return db.WithContext(ctx).Create(&outbox).Error
}
