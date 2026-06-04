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
