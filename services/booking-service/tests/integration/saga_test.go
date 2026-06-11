package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func publishTestKafkaEvent(t *testing.T, brokers []string, topic string, eventID string, eventType string, bookingID string) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokers[0], topic, 0)
	if err != nil {
		t.Fatalf("failed to dial kafka leader: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ce := map[string]interface{}{
		"specversion": "1.0",
		"id":          eventID,
		"source":      "test-service",
		"type":        eventType,
		"time":        time.Now().Format(time.RFC3339),
		"data": map[string]interface{}{
			"bookingId": bookingID,
		},
	}

	payload, err := json.Marshal(ce)
	if err != nil {
		t.Fatalf("failed to marshal cloud event: %v", err)
	}

	_, err = conn.WriteMessages(kafka.Message{
		Key:   []byte(eventID),
		Value: payload,
	})
	if err != nil {
		t.Fatalf("failed to write kafka message: %v", err)
	}
}

func TestSagaLifecycle_E2E_Kafka(t *testing.T) {
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		t.Skip("skipping SAGA E2E Kafka integration test: KAFKA_BROKERS not set")
	}

	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:postgres@localhost:5433/booking_test_db?sslmode=disable"
	}

	// 1. Database connection & preparation
	db, err := gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	truncateTables(db)
	defer truncateTables(db)

	// 2. Initialize Repositories and Coordinator
	bookingRepo := persistence.NewBookingRepository(db)
	sagaRepo := persistence.NewBookingSagaRepo(db)
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	// Injecting dummy dispute handler
	disputeHandler := command.NewDisputeBookingHandler(bookingRepo)
	resolveHandler := command.NewResolveBookingHandler(bookingRepo)

	// 3. Start Kafka Consumer in the background
	connCfg := messaging.KafkaConnConfig{
		Brokers: kafkaBrokers,
	}

	financeTopic := "finance.events.test"
	interactionTopic := "interaction.events.test"
	disputeTopic := "dispute.events.test"

	consumer := messaging.NewKafkaConsumer(
		connCfg,
		"booking-saga-test-group-"+uuid.New().String(), // Unique group ID to avoid offset caching
		[]string{financeTopic, interactionTopic, disputeTopic},
		coordinator,
		disputeHandler,
		resolveHandler,
		db,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)

	// Give the consumer a moment to join the consumer group and partition rebalance
	time.Sleep(3 * time.Second)

	// 4. Setup a Booking and BookingAcceptSaga in the DB
	clientID, _ := vo.NewClientID("00000000-0000-0000-e2e0-000000000001")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-e2e0-000000000002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	booking := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)

	if err := bookingRepo.Save(context.Background(), booking); err != nil {
		t.Fatalf("failed to save test booking: %v", err)
	}

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid.String(), now)
	saga.UpdateState(vo.SagaStateWaitingForEscrow, now)
	if err := sagaRepo.Save(context.Background(), saga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	// 5. Emit CoinEscrowed from Finance Service mock
	event1ID := uuid.New().String()
	publishTestKafkaEvent(t, []string{kafkaBrokers}, financeTopic, event1ID, "finance.coin-escrowed.v1", bid.String())

	// 6. Assert Saga state updates to WAITING_FOR_CHAT in DB
	err = pollDatabaseAssertion(50, 200*time.Millisecond, func() error {
		updatedSaga, err := sagaRepo.FindByBookingID(context.Background(), bid.String())
		if err != nil {
			return err
		}
		if updatedSaga.State != vo.SagaStateWaitingForChat {
			return fmt.Errorf("expected saga state %s, got %s", vo.SagaStateWaitingForChat, updatedSaga.State)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("SAGA failed to transition to WAITING_FOR_CHAT: %v", err)
	}

	// 7. Emit ChatRoomCreated from Interaction Service mock
	event2ID := uuid.New().String()
	publishTestKafkaEvent(t, []string{kafkaBrokers}, interactionTopic, event2ID, "interaction.chat-room-created.v1", bid.String())

	// 8. Assert Saga updates to COMPLETED and Booking updates to ACCEPTED
	err = pollDatabaseAssertion(50, 200*time.Millisecond, func() error {
		updatedSaga, err := sagaRepo.FindByBookingID(context.Background(), bid.String())
		if err != nil {
			return err
		}
		if updatedSaga.State != vo.SagaStateCompleted {
			return fmt.Errorf("expected saga state %s, got %s", vo.SagaStateCompleted, updatedSaga.State)
		}
		updatedBooking, err := bookingRepo.FindByID(context.Background(), bid)
		if err != nil {
			return err
		}
		if updatedBooking.Status() != vo.StatusAccepted {
			return fmt.Errorf("expected booking status %s, got %s", vo.StatusAccepted, updatedBooking.Status())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("SAGA failed to complete successfully: %v", err)
	}
}

func pollDatabaseAssertion(maxRetries int64, interval time.Duration, assertion func() error) error {
	var lastErr error
	for i := int64(0); i < maxRetries; i++ {
		err := assertion()
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(interval)
	}
	return fmt.Errorf("assertion failed after %d retries: %w", maxRetries, lastErr)
}
