package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

// MessagePublisher is the interface for publishing CloudEvents to a topic.
type MessagePublisher interface {
	PublishEvent(ctx context.Context, topic string, event CloudEvent) error
}

// OutboxWorker polls the outbox table and publishes unpublished events to Kafka.
type OutboxWorker struct {
	db              *gorm.DB
	publisher       MessagePublisher
	pollingInterval time.Duration
	batchSize       int64
	topic           string
}

func NewOutboxWorker(
	db *gorm.DB,
	publisher MessagePublisher,
	pollingInterval time.Duration,
	batchSize int64,
	topic string,
) *OutboxWorker {
	return &OutboxWorker{
		db:              db,
		publisher:       publisher,
		pollingInterval: pollingInterval,
		batchSize:       batchSize,
		topic:           topic,
	}
}

// Start begins the polling loop. Blocks until ctx is cancelled.
func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollingInterval)
	defer ticker.Stop()

	log.Printf("[OUTBOX] Worker started, polling every %v, topic=%s", w.pollingInterval, w.topic)

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

	// --- SQL Log Toggle for Outbox Polling ---
	// Default: SILENT (highly recommended to avoid polluting your terminal with SELECT spam)
	db := w.db.Session(&gorm.Session{Logger: w.db.Logger.LogMode(logger.Silent)}).WithContext(ctx)
	// To see Outbox SQL logs, comment the line above and uncomment the line below:
	// db := w.db.WithContext(ctx)

	err := db.
		Where("published = ?", false).
		Order("created_at asc").
		Limit(int(w.batchSize)).
		Find(&events).Error
	if err != nil {
		log.Printf("[OUTBOX] Failed to fetch unpublished events: %v", err)
		return
	}
	if len(events) == 0 {
		return
	}

	for _, evt := range events {
		if err := w.publishEvent(ctx, evt); err != nil {
			log.Printf("[OUTBOX] Failed to publish event id=%s type=%s: %v", evt.ID, evt.EventType, err)
			continue // do not mark failed events as published; they will be retried
		}

		now := time.Now()
		if err := db.
			Model(&persistence.OutboxModel{}).
			Where("id = ?", evt.ID).
			Updates(map[string]interface{}{
				"published":    true,
				"published_at": now,
			}).Error; err != nil {
			log.Printf("[OUTBOX] Failed to mark event id=%s as published: %v", evt.ID, err)
		}
	}
}

func (w *OutboxWorker) publishEvent(ctx context.Context, model persistence.OutboxModel) error {
	var rawData interface{}
	if err := json.Unmarshal([]byte(model.Payload), &rawData); err != nil {
		return err
	}

	ce := CloudEvent{
		SpecVersion:     "1.0",
		ID:              model.ID,
		Source:          "/rent-a-girlfriend/booking-service",
		Type:            model.EventType,
		DataContentType: "application/json",
		Time:            model.CreatedAt,
		Data:            rawData,
	}

	return w.publisher.PublishEvent(ctx, w.topic, ce)
}
