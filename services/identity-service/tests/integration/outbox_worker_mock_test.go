package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/tests/testhelper"
)

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishEvent(ctx context.Context, topic string, key string, event cloudevents.Event) error {
	args := m.Called(ctx, topic, key, event)
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
		mock.Anything,
		mock.MatchedBy(func(ev cloudevents.Event) bool {
			return ev.ID() == eventID.String() && ev.Type() == "test.event.v1"
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
		"identity.events",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish event qua outbox
	testUserID := "e2e-user-" + uuid.New().String()
	testEvent := event.UserRegistered{
		UserRegisteredPayload: &identityv1.UserRegisteredPayload{
			UserId:    testUserID,
			Email:     "e2e-mock@test.com",
			Role:      "CLIENT",
			GoogleId:  "google-" + uuid.New().String(),
			Timestamp: timestamppb.New(time.Now().UTC().Truncate(time.Second)),
			Name:      "E2E Mock User",
		},
	}
	err := outboxPublisher.Publish(ctx, testEvent)
	require.NoError(t, err)

	// Xác định eventID từ DB để setup mock expectation chính xác
	var entry persistence.OutboxModel
	err = db.Where("CAST(payload AS TEXT) LIKE ?", "%"+testUserID+"%").First(&entry).Error
	require.NoError(t, err)

	mockKafka.On("PublishEvent",
		mock.Anything,
		"identity.events",
		mock.Anything,
		mock.MatchedBy(func(ev cloudevents.Event) bool {
			return ev.ID() == entry.ID.String() && ev.Type() == testEvent.EventType()
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

func TestOutboxWorker_ConcurrencyRaceCondition(t *testing.T) {
	db := testhelper.StartPostgresContainer(t)

	// Xóa sạch outbox events để tránh nhiễu
	err := db.Exec("DELETE FROM outbox_events").Error
	require.NoError(t, err)

	mockKafka := new(MockPublisher)

	// Tạo 1 event chưa publish
	eventID := uuid.New()
	testPayload := `{"userId":"user-concurrent-123","email":"concurrent@test.com"}`
	err = db.Create(&persistence.OutboxModel{
		ID:        eventID,
		EventType: "test.event.v1",
		Payload:   testPayload,
		Published: false,
		CreatedAt: time.Now(),
	}).Error
	require.NoError(t, err)

	// Setup mock expectation: Chỉ được phép publish duy nhất 1 lần
	mockKafka.On("PublishEvent",
		mock.Anything,
		"test-topic",
		mock.Anything,
		mock.MatchedBy(func(ev cloudevents.Event) bool {
			return ev.ID() == eventID.String() && ev.Type() == "test.event.v1"
		}),
	).Return(nil)

	// Chạy đồng thời 3 workers cùng kéo và xử lý
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// Sử dụng polling interval cực ngắn để kích hoạt xử lý nhanh
			worker := messaging.NewOutboxWorker(
				db,
				mockKafka,
				10*time.Millisecond,
				10,
				"test-topic",
			)
			worker.Start(ctx)
		}(i)
	}

	// Đợi các worker chạy và hoàn thành xử lý
	time.Sleep(1 * time.Second)
	cancel() // Dừng tất cả worker
	wg.Wait()

	// Assertions: mockKafka.PublishEvent chỉ được gọi đúng 1 lần
	mockKafka.AssertNumberOfCalls(t, "PublishEvent", 1)

	// Đảm bảo trạng thái trong database là đã được published
	var updated persistence.OutboxModel
	err = db.Where("id = ?", eventID).First(&updated).Error
	require.NoError(t, err)
	assert.True(t, updated.Published)
	assert.NotNil(t, updated.PublishedAt)
	assert.Nil(t, updated.LockedUntil)
}
