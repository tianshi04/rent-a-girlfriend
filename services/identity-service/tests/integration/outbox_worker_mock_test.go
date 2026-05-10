package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rent-a-girlfriend/identity-service/internal/bootstrap"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

// MockPublisher is a mock implementation of messaging.MessagePublisher
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishEvent(ctx context.Context, topic string, event messaging.CloudEvent) error {
	args := m.Called(ctx, topic, event)
	return args.Error(0)
}

func TestOutboxWorker_WithMockKafka(t *testing.T) {
	if testing.Short() {
		t.Skip("Bỏ qua integration test trong chế độ short mode")
	}

	// 1. Setup
	cfg := bootstrap.LoadConfig()
	db, err := bootstrap.InitDatabase(cfg.Database)
	require.NoError(t, err)

	db.Exec("DELETE FROM outbox_events")

	mockKafka := new(MockPublisher)
	worker := messaging.NewOutboxWorker(
		db,
		mockKafka,
		100*time.Millisecond,
		10,
		"test-topic",
	)

	// 2. Insert some unpublished events directly into DB
	eventID := uuid.New()
	testPayload := `{"userId":"user-mock-123","email":"mock@test.com"}`
	db.Create(&persistence.OutboxModel{
		ID:        eventID,
		EventType: "test.event.v1",
		Payload:   testPayload,
		Published: false,
		CreatedAt: time.Now(),
	})

	// 3. Setup mock expectations
	mockKafka.On("PublishEvent", mock.Anything, "test-topic", mock.MatchedBy(func(ev messaging.CloudEvent) bool {
		return ev.ID == eventID.String() && ev.Type == "test.event.v1"
	})).Return(nil)

	// 4. Run worker briefly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go worker.Start(ctx)

	// Wait for worker to process
	time.Sleep(500 * time.Millisecond)

	// 5. Assertions
	mockKafka.AssertExpectations(t)

	// Check if marked as published in DB
	var entry persistence.OutboxModel
	err = db.Where("id = ?", eventID).First(&entry).Error
	require.NoError(t, err)
	assert.True(t, entry.Published, "Worker phải đánh dấu sự kiện là đã gửi trong DB")
}
