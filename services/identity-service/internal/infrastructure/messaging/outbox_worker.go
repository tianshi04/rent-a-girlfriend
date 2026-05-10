package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

type MessagePublisher interface {
	PublishEvent(ctx context.Context, topic string, event CloudEvent) error
}

type OutboxWorker struct {
	db              *gorm.DB
	publisher       MessagePublisher
	pollingInterval time.Duration
	batchSize       int
	topic           string
	serviceSource   string
}

func NewOutboxWorker(
	db *gorm.DB,
	publisher MessagePublisher,
	pollingInterval time.Duration,
	batchSize int,
	topic string,
) *OutboxWorker {
	return &OutboxWorker{
		db:              db,
		publisher:       publisher,
		pollingInterval: pollingInterval,
		batchSize:       batchSize,
		topic:           topic,
		serviceSource:   "/rent-a-gf/identity-service",
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollingInterval)
	defer ticker.Stop()

	log.Printf("[OUTBOX] Worker started with polling interval %v", w.pollingInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("[OUTBOX] Worker stopping...")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	var events []persistence.OutboxModel

	// Find unpublished events
	err := w.db.WithContext(ctx).
		Where("published = ?", false).
		Order("created_at asc").
		Limit(w.batchSize).
		Find(&events).Error

	if err != nil {
		log.Printf("[OUTBOX] Failed to fetch events: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	for _, evt := range events {
		err := w.publishEvent(ctx, evt)
		if err != nil {
			log.Printf("[OUTBOX] Failed to publish event %s: %v", evt.ID, err)
			continue
		}

		// Mark as published
		err = w.db.WithContext(ctx).
			Model(&persistence.OutboxModel{}).
			Where("id = ?", evt.ID).
			Update("published", true).Error

		if err != nil {
			log.Printf("[OUTBOX] Failed to mark event %s as published: %v", evt.ID, err)
		}
	}
}

func (w *OutboxWorker) publishEvent(ctx context.Context, model persistence.OutboxModel) error {
	var rawData interface{}
	err := json.Unmarshal([]byte(model.Payload), &rawData)
	if err != nil {
		return err
	}

	cloudEvent := CloudEvent{
		SpecVersion:     "1.0",
		ID:              model.ID.String(),
		Source:          w.serviceSource,
		Type:            model.EventType,
		DataContentType: "application/json",
		Time:            model.CreatedAt,
		Data:            rawData,
	}

	return w.publisher.PublishEvent(ctx, w.topic, cloudEvent)
}
