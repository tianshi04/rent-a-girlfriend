package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/application/port"
	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
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

func TestE2E_GetBooking_SecurityMatrix(t *testing.T) {
	repo := &e2eMockBookingRepository{}

	// Setup a sample booking
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000111")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000222")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	repo.booking = b

	// Application handlers
	getBookingHandler := query.NewGetBookingHandler(repo)
	listBookingsHandler := query.NewListBookingsHandler(repo)

	// Wire-up gRPC Handler
	grpcHandler := handler.NewBookingGRPCHandler(
		nil, nil, nil, nil, nil,
		getBookingHandler,
		listBookingsHandler,
	)

	ts, cleanup := startE2ETestServer(t, grpcHandler)
	defer cleanup()

	// 1. Test Client access to own booking -> OK (200)
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/bookings/"+bid.String(), nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000111")
	req.Header.Set("user-role", "CLIENT")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for owner client, got %d", resp.StatusCode)
	}

	// 2. Test Other Client access to booking -> Forbidden (403)
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/bookings/"+bid.String(), nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000999")
	req.Header.Set("user-role", "CLIENT")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for different client, got %d", resp.StatusCode)
	}

	// 3. Test Admin access to booking -> OK (200)
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/bookings/"+bid.String(), nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("user-role", "ADMIN")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for admin, got %d", resp.StatusCode)
	}
}

func TestE2E_ListBookings_RoleFilters(t *testing.T) {
	repo := &e2eMockBookingRepository{}

	// Setup a sample booking
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000111")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000222")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	repo.booking = b

	listBookingsHandler := query.NewListBookingsHandler(repo)
	grpcHandler := handler.NewBookingGRPCHandler(
		nil, nil, nil, nil, nil,
		nil,
		listBookingsHandler,
	)

	ts, cleanup := startE2ETestServer(t, grpcHandler)
	defer cleanup()

	// 1. Client views list -> Should map callerID to clientID parameter
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/bookings?view=pending", nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000111")
	req.Header.Set("user-role", "CLIENT")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var listResp struct {
		Bookings []struct {
			BookingId string `json:"bookingId"`
		} `json:"bookings"`
	}
	_ = json.Unmarshal(body, &listResp)

	if len(listResp.Bookings) != 1 || listResp.Bookings[0].BookingId != bid.String() {
		t.Errorf("expected 1 booking in list, got %+v", listResp)
	}
}

func TestE2E_RequestBooking_SuccessAndFailure(t *testing.T) {
	repo := &e2eMockBookingRepository{}
	
	// Mock Profile snapshot
	price := vo.MustMoney(600)
	snap, _ := vo.NewScenarioSnapshot(price, 90)
	profileSvc := &e2eMockProfileService{snapshot: &snap}

	// Finance Service starts with success (nil errors)
	financeSvc := &e2eMockFinanceService{}

	// Setup application command handlers
	requestBookingHandler := command.NewRequestBookingHandler(repo, profileSvc, financeSvc)
	
	// Wire-up gRPC Handler
	grpcHandler := handler.NewBookingGRPCHandler(
		requestBookingHandler, nil, nil, nil, nil, nil, nil,
	)

	ts, cleanup := startE2ETestServer(t, grpcHandler)
	defer cleanup()

	// --- 1. SUCCESS PATH ---
	reqBody := map[string]interface{}{
		"clientId":    "00000000-0000-0000-0000-000000000123",
		"companionId": "00000000-0000-0000-0000-000000000456",
		"scenarioId":  "00000000-0000-0000-0000-000000000789",
		"startTime":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/bookings", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000123")
	req.Header.Set("user-role", "CLIENT")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	// gRPC Gateway maps success to 200 OK
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for request booking, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var successResp struct {
		BookingId string `json:"bookingId"`
		Status    string `json:"status"`
		Message   string `json:"message"`
	}
	_ = json.Unmarshal(body, &successResp)

	if successResp.BookingId == "" {
		t.Error("expected a non-empty booking ID")
	}
	if successResp.Status != "BOOKING_STATUS_PENDING" {
		t.Errorf("expected booking status BOOKING_STATUS_PENDING, got %s", successResp.Status)
	}

	// Verify that the booking is saved in repository
	if repo.booking == nil || repo.booking.ID().String() != successResp.BookingId {
		t.Error("booking was not saved in repository")
	}

	// --- 2. FAILURE PATH: Insufficient Funds ---
	financeSvc.freezeErr = domainerr.ErrInsufficientFunds
	repo.booking = nil // Reset repository

	bodyBytes, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", ts.URL+"/api/v1/bookings", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000123")
	req.Header.Set("user-role", "CLIENT")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	// gRPC-Gateway maps FailedPrecondition to 400 Bad Request
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request when funds insufficient, got %d", resp.StatusCode)
	}

	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	var errorResp struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &errorResp)

	if errorResp.Message != domainerr.ErrInsufficientFunds.Error() {
		t.Errorf("expected error message '%s', got %s", domainerr.ErrInsufficientFunds.Error(), errorResp.Message)
	}
	if repo.booking != nil {
		t.Error("booking should not have been saved in repository on failure")
	}
}

func TestE2E_AcceptBooking_Success(t *testing.T) {
	repo := &e2eMockBookingRepository{}
	sagaRepo := &e2eMockBookingSagaRepository{}
	outbox := &e2eMockOutboxPublisher{}

	// Pre-populate a PENDING booking in repository
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000123")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000456")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	repo.booking = b

	// Create AcceptBookingHandler with a nil *gorm.DB — safe because outbox is mocked
	acceptBookingHandler := command.NewAcceptBookingHandler(repo, sagaRepo, nil, outbox)

	// Wire-up gRPC Handler
	grpcHandler := handler.NewBookingGRPCHandler(
		nil, acceptBookingHandler, nil, nil, nil, nil, nil,
	)

	ts, cleanup := startE2ETestServer(t, grpcHandler)
	defer cleanup()

	// Call AcceptBooking PUT /api/v1/bookings/:id/accept
	reqBody := map[string]interface{}{
		"companionId": "00000000-0000-0000-0000-000000000456",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/bookings/"+bid.String()+"/accept", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000456")
	req.Header.Set("user-role", "COMPANION")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for accept booking, got %d", resp.StatusCode)
	}

	// Verify that the Accept SAGA was created
	if sagaRepo.saga == nil || sagaRepo.saga.BookingID != bid.String() {
		t.Error("expected accept saga to be created and saved")
	}
}

func TestE2E_RejectBooking_Success(t *testing.T) {
	repo := &e2eMockBookingRepository{}
	financeSvc := &e2eMockFinanceService{}

	// Pre-populate a PENDING booking in repository
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000123")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000456")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	repo.booking = b

	// Create RejectBookingHandler
	rejectBookingHandler := command.NewRejectBookingHandler(repo, financeSvc)

	// Wire-up gRPC Handler
	grpcHandler := handler.NewBookingGRPCHandler(
		nil, nil, rejectBookingHandler, nil, nil, nil, nil,
	)

	ts, cleanup := startE2ETestServer(t, grpcHandler)
	defer cleanup()

	// Call RejectBooking PUT /api/v1/bookings/:id/reject
	reqBody := map[string]interface{}{
		"companionId": "00000000-0000-0000-0000-000000000456",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/bookings/"+bid.String()+"/reject", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000456")
	req.Header.Set("user-role", "COMPANION")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for reject booking, got %d", resp.StatusCode)
	}

	// Verify that the booking status was updated to REJECTED in repository
	if repo.booking == nil || repo.booking.Status() != vo.StatusRejected {
		t.Errorf("expected booking status to be REJECTED, got %s", repo.booking.Status())
	}
}

func TestE2E_CancelBooking_Success(t *testing.T) {
	repo := &e2eMockBookingRepository{}

	// Pre-populate an ACCEPTED booking in repository
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000123")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000456")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusAccepted, "", false, 1, now, now)
	repo.booking = b

	// Create CancelBookingHandler
	cancelBookingHandler := command.NewCancelBookingHandler(repo)

	// Wire-up gRPC Handler
	grpcHandler := handler.NewBookingGRPCHandler(
		nil, nil, nil, cancelBookingHandler, nil, nil, nil,
	)

	ts, cleanup := startE2ETestServer(t, grpcHandler)
	defer cleanup()

	// Call CancelBooking PUT /api/v1/bookings/:id/cancel
	reqBody := map[string]interface{}{
		"actorId": "00000000-0000-0000-0000-000000000123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/bookings/"+bid.String()+"/cancel", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000123")
	req.Header.Set("user-role", "CLIENT")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for cancel booking, got %d", resp.StatusCode)
	}

	// Verify that the booking status was updated to CANCELLED in repository
	if repo.booking == nil || repo.booking.Status() != vo.StatusCancelled {
		t.Errorf("expected booking status to be CANCELLED, got %s", repo.booking.Status())
	}
}
