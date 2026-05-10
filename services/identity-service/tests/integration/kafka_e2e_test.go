package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rent-a-girlfriend/identity-service/internal/bootstrap"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

func TestKafkaOutbox_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Bỏ qua E2E test trong chế độ short mode")
	}

	// 1. Setup
	cfg := bootstrap.LoadConfig()
	db, err := bootstrap.InitDatabase(cfg.Database)
	require.NoError(t, err)

	db.Exec("DELETE FROM outbox_events")

	kafkaAdapter := messaging.NewKafkaAdapter(cfg.Kafka.Brokers)
	defer kafkaAdapter.Close()

	outboxPublisher := persistence.NewOutboxPublisher(db)
	worker := messaging.NewOutboxWorker(
		db,
		kafkaAdapter,
		200*time.Millisecond,
		10,
		cfg.Kafka.TopicIdentity,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Publish Event
	testUserID := "e2e-user-" + uuid.New().String()
	testEvent := event.UserRegistered{
		UserID:    testUserID,
		Email:     "e2e@test.com",
		Role:      "CLIENT",
		GoogleID:  "google-" + uuid.New().String(),
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}

	err = outboxPublisher.Publish(ctx, testEvent)
	require.NoError(t, err)

	// 3. Start Worker
	workerCtx, workerCancel := context.WithTimeout(ctx, 5*time.Second)
	defer workerCancel()
	go worker.Start(workerCtx)

	// 4. Verification in Kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(cfg.Kafka.Brokers, ","),
		Topic:    cfg.Kafka.TopicIdentity,
		GroupID:  "e2e-test-group-" + uuid.New().String(),
		MaxWait:  5 * time.Second,
	})
	defer reader.Close()

	found := false
	timeout := time.After(15 * time.Second)
	for !found {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for message in Kafka (E2E)")
		default:
			msg, err := reader.ReadMessage(ctx)
			require.NoError(t, err)

			var cloudEvent messaging.CloudEvent
			err = json.Unmarshal(msg.Value, &cloudEvent)
			if err != nil {
				continue
			}

			dataMap, ok := cloudEvent.Data.(map[string]interface{})
			if !ok {
				continue
			}

			if dataMap["userId"] == testUserID {
				found = true
				assert.Equal(t, testEvent.EventType(), cloudEvent.Type)
				assert.Equal(t, testEvent.Email, dataMap["email"])
			}
		}
	}

	// 5. Final DB Check
	var entry persistence.OutboxModel
	err = db.Where("CAST(payload AS TEXT) LIKE ?", "%"+testUserID+"%").First(&entry).Error
	require.NoError(t, err)
	assert.True(t, entry.Published, "DB row must be marked as published")
}
