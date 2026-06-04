package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	handler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/handler"
)

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
