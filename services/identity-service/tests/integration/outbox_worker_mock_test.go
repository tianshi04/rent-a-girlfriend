package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/tests/testhelper"
)

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishEvent(ctx context.Context, topic string, event messaging.CloudEvent) error {
	args := m.Called(ctx, topic, event)
	return args.Error(0)
}

// TestOutboxWorker_WithMockKafka kiểm tra OutboxWorker xử lý đúng các event
// trong DB và gọi MockPublisher. Không cần Kafka broker thật.
func TestOutboxWorker_WithMockKafka(t *testing.T) {
	db := testhelper.StartPostgresContainer(t)

	mockKafka := new(MockPublisher)
	worker := messaging.NewOutboxWorker(
		db,
		mockKafka,
		100*time.Millisecond,
		10,
		"test-topic",
	)

	// Insert event chưa publish vào DB
	eventID := uuid.New()
	testPayload := `{"userId":"user-mock-123","email":"mock@test.com"}`
	err := db.Create(&persistence.OutboxModel{
		ID:        eventID,
		EventType: "test.event.v1",
		Payload:   testPayload,
		Published: false,
		CreatedAt: time.Now(),
	}).Error
	require.NoError(t, err)

	// Setup mock expectation
	mockKafka.On("PublishEvent",
		mock.Anything,
		"test-topic",
		mock.MatchedBy(func(ev messaging.CloudEvent) bool {
			return ev.ID == eventID.String() && ev.Type == "test.event.v1"
		}),
	).Return(nil)

	// Chạy worker trong giới hạn thời gian
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	go worker.Start(ctx)

	time.Sleep(500 * time.Millisecond)

	// Assertions
	mockKafka.AssertExpectations(t)

	var entry persistence.OutboxModel
	err = db.Where("id = ?", eventID).First(&entry).Error
	require.NoError(t, err)
	assert.True(t, entry.Published, "Worker phải đánh dấu sự kiện là đã gửi")
}

// TestKafkaOutbox_E2E kiểm tra luồng Outbox → Worker → MockPublisher end-to-end.
// Không cần Kafka broker thật — dùng mock để xác nhận CloudEvent được publish đúng.
func TestKafkaOutbox_WithMockBroker(t *testing.T) {
	db := testhelper.StartPostgresContainer(t)

	mockKafka := new(MockPublisher)
	outboxPublisher := persistence.NewOutboxPublisher(db)
	worker := messaging.NewOutboxWorker(
		db,
		mockKafka,
		200*time.Millisecond,
		10,
		"identity-events",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish event qua outbox
	testUserID := "e2e-user-" + uuid.New().String()
	testEvent := event.UserRegistered{
		UserID:    testUserID,
		Email:     "e2e-mock@test.com",
		Role:      "CLIENT",
		GoogleID:  "google-" + uuid.New().String(),
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}
	err := outboxPublisher.Publish(ctx, testEvent)
	require.NoError(t, err)

	// Xác định eventID từ DB để setup mock expectation chính xác
	var entry persistence.OutboxModel
	err = db.Where("CAST(payload AS TEXT) LIKE ?", "%"+testUserID+"%").First(&entry).Error
	require.NoError(t, err)

	mockKafka.On("PublishEvent",
		mock.Anything,
		"identity-events",
		mock.MatchedBy(func(ev messaging.CloudEvent) bool {
			return ev.ID == entry.ID.String() && ev.Type == testEvent.EventType()
		}),
	).Return(nil)

	// Chạy worker
	workerCtx, workerCancel := context.WithTimeout(ctx, 3*time.Second)
	defer workerCancel()
	go worker.Start(workerCtx)
	time.Sleep(1 * time.Second)

	// Xác nhận mock được gọi và DB được mark published
	mockKafka.AssertExpectations(t)

	var updated persistence.OutboxModel
	err = db.Where("id = ?", entry.ID).First(&updated).Error
	require.NoError(t, err)
	assert.True(t, updated.Published, "DB row phải được mark là published")
}
