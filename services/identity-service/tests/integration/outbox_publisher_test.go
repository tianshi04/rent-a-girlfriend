package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rent-a-girlfriend/identity-service/internal/bootstrap"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

func TestOutboxPublisher_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Bỏ qua integration test trong chế độ short mode")
	}

	// 1. Setup
	cfg := bootstrap.LoadConfig()
	db, err := bootstrap.InitDatabase(cfg.Database)
	require.NoError(t, err)

	// Clean up
	db.Exec("DELETE FROM outbox_events")

	publisher := persistence.NewOutboxPublisher(db)
	ctx := context.Background()

	// 2. Publish a test event
	testUserID := uuid.New().String()
	testEvent := event.UserRegistered{
		UserID:    testUserID,
		Email:     "integration@test.com",
		Role:      "CLIENT",
		GoogleID:  "google-" + uuid.New().String(),
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}

	err = publisher.Publish(ctx, testEvent)
	require.NoError(t, err)

	// 3. Verify in DB
	var entry persistence.OutboxModel
	err = db.Where("CAST(payload AS TEXT) LIKE ?", "%"+testUserID+"%").First(&entry).Error
	require.NoError(t, err)

	assert.Equal(t, testEvent.EventType(), entry.EventType)
	assert.False(t, entry.Published, "Sự kiện mới phải có published = false")
	assert.NotNil(t, entry.ID)
	assert.WithinDuration(t, time.Now(), entry.CreatedAt, 2*time.Second)

	// Verify payload content
	assert.Contains(t, entry.Payload, testUserID)
	assert.Contains(t, entry.Payload, "integration@test.com")
}
