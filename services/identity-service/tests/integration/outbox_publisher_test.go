package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/tests/testhelper"
)

func TestOutboxPublisher_Integration(t *testing.T) {
	db := testhelper.StartPostgresContainer(t)

	testhelper.WithTx(t, db, func(tx *gorm.DB) {
		publisher := persistence.NewOutboxPublisher(tx)
		ctx := context.Background()

		testUserID := uuid.New().String()
		testEvent := event.UserRegistered{
			UserRegisteredPayload: &identityv1.UserRegisteredPayload{
				UserId:    testUserID,
				Email:     "integration@test.com",
				Role:      "CLIENT",
				GoogleId:  "google-" + uuid.New().String(),
				Timestamp: timestamppb.New(time.Now().UTC().Truncate(time.Second)),
				Name:      "Integration Test User",
			},
		}

		err := publisher.Publish(ctx, testEvent)
		require.NoError(t, err)

		var entry persistence.OutboxModel
		err = tx.Where("CAST(payload AS TEXT) LIKE ?", "%"+testUserID+"%").First(&entry).Error
		require.NoError(t, err)

		assert.Equal(t, testEvent.EventType(), entry.EventType)
		assert.False(t, entry.Published, "Sự kiện mới phải có published = false")
		assert.NotEqual(t, uuid.Nil, entry.ID)
		assert.WithinDuration(t, time.Now(), entry.CreatedAt, 2*time.Second)
		assert.Contains(t, entry.Payload, testUserID)
		assert.Contains(t, entry.Payload, "integration@test.com")
	})
	// Transaction đã được ROLLBACK tự động — không có dữ liệu bẩn
}
