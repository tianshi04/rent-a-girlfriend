package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	handler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/handler"
)

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
