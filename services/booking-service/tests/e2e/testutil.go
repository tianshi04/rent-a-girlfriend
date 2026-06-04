package e2e

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	"github.com/rent-a-girlfriend/booking-service/internal/application/port"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	handler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/handler"
	router "github.com/rent-a-girlfriend/booking-service/internal/interfaces/http"
)

// In-memory mock repositories that behave predictably for E2E
type e2eMockBookingRepository struct {
	booking *aggregate.Booking
}

func (m *e2eMockBookingRepository) Save(ctx context.Context, b *aggregate.Booking) error {
	m.booking = b
	return nil
}

func (m *e2eMockBookingRepository) Update(ctx context.Context, b *aggregate.Booking) error {
	m.booking = b
	return nil
}

func (m *e2eMockBookingRepository) FindByID(ctx context.Context, id vo.BookingID) (*aggregate.Booking, error) {
	if m.booking != nil && m.booking.ID() == id {
		return m.booking, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *e2eMockBookingRepository) CountPendingByCompanion(ctx context.Context, companionID vo.CompanionID) (int64, error) {
	return 0, nil
}

func (m *e2eMockBookingRepository) HasOverlappingBooking(ctx context.Context, actorID string, isCompanion bool, statuses []vo.BookingStatus, startTime, endTime time.Time) (bool, error) {
	return false, nil
}

func (m *e2eMockBookingRepository) FindByFilters(ctx context.Context, filters repository.BookingFilters) ([]*aggregate.Booking, int64, error) {
	var list []*aggregate.Booking
	if m.booking != nil {
		list = append(list, m.booking)
	}
	return list, int64(len(list)), nil
}

func (m *e2eMockBookingRepository) FindAcceptedBookingsPastEndTimeBuffer(ctx context.Context, now time.Time, buffer time.Duration) ([]*aggregate.Booking, error) {
	return nil, nil
}

func (m *e2eMockBookingRepository) FindPendingBookingsEligibleForTimeout(ctx context.Context, now time.Time) ([]*aggregate.Booking, error) {
	return nil, nil
}

// In-memory mock Profile Service for E2E
type e2eMockProfileService struct {
	snapshot *vo.ScenarioSnapshot
	err      error
}

func (m *e2eMockProfileService) GetScenarioSnapshot(ctx context.Context, scenarioID string) (*vo.ScenarioSnapshot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.snapshot, nil
}

// In-memory mock Finance Service for E2E
type e2eMockFinanceService struct {
	freezeErr   error
	unfreezeErr error
}

func (m *e2eMockFinanceService) FreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	return m.freezeErr
}

func (m *e2eMockFinanceService) UnfreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	return m.unfreezeErr
}

// In-memory mock Saga Repository for E2E
type e2eMockBookingSagaRepository struct {
	saga *aggregate.BookingAcceptSaga
}

func (m *e2eMockBookingSagaRepository) Save(ctx context.Context, saga *aggregate.BookingAcceptSaga) error {
	m.saga = saga
	return nil
}

func (m *e2eMockBookingSagaRepository) Update(ctx context.Context, saga *aggregate.BookingAcceptSaga) error {
	m.saga = saga
	return nil
}

func (m *e2eMockBookingSagaRepository) FindByID(ctx context.Context, id string) (*aggregate.BookingAcceptSaga, error) {
	if m.saga != nil && m.saga.ID == id {
		return m.saga, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *e2eMockBookingSagaRepository) FindByBookingID(ctx context.Context, bookingID string) (*aggregate.BookingAcceptSaga, error) {
	if m.saga != nil && m.saga.BookingID == bookingID {
		return m.saga, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// e2eMockOutboxPublisher is a no-op EventPublisher for E2E tests.
// It satisfies port.EventPublisher without requiring a real database connection.
type e2eMockOutboxPublisher struct{}

func (m *e2eMockOutboxPublisher) Publish(_ context.Context, _ event.DomainEvent) error {
	return nil
}

var _ port.EventPublisher = (*e2eMockOutboxPublisher)(nil)

func startE2ETestServer(t *testing.T, grpcHandler *handler.BookingGRPCHandler) (*httptest.Server, func()) {
	// 1. Listen on a random localhost port for gRPC
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on random port: %v", err)
	}
	grpcAddr := lis.Addr().String()

	// 2. Start gRPC server
	gServer := grpc.NewServer()
	bookingv1.RegisterBookingServiceServer(gServer, grpcHandler)

	go func() {
		_ = gServer.Serve(lis)
	}()

	// 3. Start gRPC Gateway HTTP Server pointing to the local gRPC server
	ctx, cancel := context.WithCancel(context.Background())
	gwMux, err := router.NewGateway(ctx, grpcAddr)
	if err != nil {
		gServer.Stop()
		cancel()
		t.Fatalf("failed to create gateway: %v", err)
	}

	ts := httptest.NewServer(gwMux)

	cleanup := func() {
		ts.Close()
		gServer.Stop()
		cancel()
	}

	return ts, cleanup
}
