package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	handler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/handler"
)

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
