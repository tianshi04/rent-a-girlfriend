package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func getTestDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/booking_test_db?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// AutoMigrate all models
	err = db.AutoMigrate(
		&persistence.BookingModel{},
		&persistence.OutboxModel{},
		&persistence.BookingAcceptSagaModel{},
		&persistence.ProcessedEventModel{},
	)
	if err != nil {
		t.Fatalf("failed to run GORM auto-migrations: %v", err)
	}

	return db
}

func truncateTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE bookings CASCADE")
	db.Exec("TRUNCATE TABLE outbox CASCADE")
	db.Exec("TRUNCATE TABLE booking_accept_sagas CASCADE")
	db.Exec("TRUNCATE TABLE processed_events CASCADE")
}

func TestBookingRepository_Integration(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)

	// Test Save
	err := repo.Save(context.Background(), b)
	if err != nil {
		t.Fatalf("failed to save booking: %v", err)
	}

	// Test FindByID
	found, err := repo.FindByID(context.Background(), bid)
	if err != nil {
		t.Fatalf("failed to find booking: %v", err)
	}
	if found.ID() != bid {
		t.Errorf("expected ID %s, got %s", bid.String(), found.ID().String())
	}

	// Test HasOverlappingBooking
	overlap, err := repo.HasOverlappingBooking(
		context.Background(),
		companionID.String(),
		true,
		[]vo.BookingStatus{vo.StatusPending},
		now.Add(4*time.Hour),
		now.Add(6*time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to check overlapping booking: %v", err)
	}
	if !overlap {
		t.Error("expected overlap to be true")
	}

	// Test CountPendingByCompanion
	count, err := repo.CountPendingByCompanion(context.Background(), companionID)
	if err != nil {
		t.Fatalf("failed to count pending bookings: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 pending booking, got %d", count)
	}

	// Test Update
	err = b.Accept(companionID, now)
	if err != nil {
		t.Fatalf("failed to accept booking: %v", err)
	}
	err = repo.Update(context.Background(), b)
	if err != nil {
		t.Fatalf("failed to update booking: %v", err)
	}

	updated, _ := repo.FindByID(context.Background(), bid)
	if updated.Status() != vo.StatusAccepted {
		t.Errorf("expected booking status %s, got %s", vo.StatusAccepted, updated.Status())
	}
}

func TestBookingSagaRepository_Integration(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingSagaRepo(db)

	bookingID := "550e8400-e29b-41d4-a716-446655440001"
	sagaID := "660e8400-e29b-41d4-a716-446655440001"
	now := time.Now()

	saga := aggregate.NewBookingAcceptSaga(sagaID, bookingID, now)
	saga.UpdateState(vo.SagaStateWaitingForEscrow, now)

	// Save saga
	err := repo.Save(context.Background(), saga)
	if err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Find by ID
	found, err := repo.FindByID(context.Background(), sagaID)
	if err != nil {
		t.Fatalf("failed to find saga by ID: %v", err)
	}
	if found.ID != sagaID {
		t.Errorf("expected saga ID %s, got %s", sagaID, found.ID)
	}

	// Find by Booking ID
	foundByBooking, err := repo.FindByBookingID(context.Background(), bookingID)
	if err != nil {
		t.Fatalf("failed to find saga by booking ID: %v", err)
	}
	if foundByBooking.ID != sagaID {
		t.Errorf("expected saga ID %s, got %s", sagaID, foundByBooking.ID)
	}

	// Update saga state
	saga.UpdateState(vo.SagaStateCompleted, now)
	err = repo.Update(context.Background(), saga)
	if err != nil {
		t.Fatalf("failed to update saga: %v", err)
	}

	updated, _ := repo.FindByID(context.Background(), sagaID)
	if updated.State != vo.SagaStateCompleted {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateCompleted, updated.State)
	}
}
