package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

type MessagePublisher interface {
	PublishEvent(ctx context.Context, topic string, key string, event cloudevents.Event) error
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

	// Wrap SELECT FOR UPDATE SKIP LOCKED and lock update in a single DB transaction to release locks immediately on commit
	err := w.db.Transaction(func(tx *gorm.DB) error {
		silentTx := tx.Session(&gorm.Session{Logger: tx.Logger.LogMode(logger.Silent)}).WithContext(ctx)

		now := time.Now()
		err := silentTx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("published = ? AND (locked_until IS NULL OR locked_until < ?)", false, now).
			Order("created_at asc").
			Limit(w.batchSize).
			Find(&events).Error
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}

		eventIDs := make([]uuid.UUID, len(events))
		for i, e := range events {
			eventIDs[i] = e.ID
		}

		// Lock for 1 minute to prevent concurrent workers from fetching the same events
		lockedUntil := now.Add(1 * time.Minute)
		return silentTx.
			Model(&persistence.OutboxModel{}).
			Where("id IN ?", eventIDs).
			Updates(map[string]interface{}{
				"locked_until": lockedUntil,
			}).Error
	})

	if err != nil {
		log.Printf("[OUTBOX] Failed to fetch and lock outbox batch: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	// Publish to Kafka outside of the DB transaction block (preventing lock holding on Kafka network delays)
	for _, evt := range events {
		err := w.publishEvent(ctx, evt)
		if err != nil {
			log.Printf("[OUTBOX] Failed to publish event %s: %v. Releasing lock to retry...", evt.ID, err)

			// Release lock immediately on failure so it can be retried in the next poll
			_ = w.db.WithContext(ctx).
				Model(&persistence.OutboxModel{}).
				Where("id = ?", evt.ID).
				Updates(map[string]interface{}{
					"locked_until": nil,
				}).Error
		} else {
			now := time.Now()
			// Mark as published and clear lock on success
			_ = w.db.WithContext(ctx).
				Model(&persistence.OutboxModel{}).
				Where("id = ?", evt.ID).
				Updates(map[string]interface{}{
					"published":    true,
					"published_at": &now,
					"locked_until": nil,
				}).Error
		}
	}
}

func (w *OutboxWorker) publishEvent(ctx context.Context, model persistence.OutboxModel) error {
	var rawData interface{}
	err := json.Unmarshal([]byte(model.Payload), &rawData)
	if err != nil {
		return err
	}

	// Extract userId from payload map for partition key
	var payloadMap map[string]interface{}
	_ = json.Unmarshal([]byte(model.Payload), &payloadMap)
	keyStr := ""
	if val, ok := payloadMap["userId"].(string); ok {
		keyStr = val
	} else if val, ok := payloadMap["user_id"].(string); ok {
		keyStr = val
	}

	event := cloudevents.NewEvent()
	event.SetID(model.ID.String())
	event.SetSource(w.serviceSource)
	event.SetType(model.EventType)
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetTime(model.CreatedAt)

	corrID := model.CorrelationID
	if corrID == "" {
		corrID = model.ID.String() // fallback to event ID
	}
	event.SetExtension("correlationid", corrID)

	if err := event.SetData(cloudevents.ApplicationJSON, rawData); err != nil {
		return err
	}

	return w.publisher.PublishEvent(ctx, w.topic, keyStr, event)
}
