package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func TestE2E_ListBookings_RoleFilters(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	// Setup a sample booking
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000111")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000222")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)

	err := repo.Save(context.Background(), b)
	if err != nil {
		t.Fatalf("failed to save booking: %v", err)
	}

	// 1. Client views list -> Should map callerID to clientID parameter
	url := fmt.Sprintf("%s/api/v1/bookings?view=pending", getBaseURL())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000111")
	req.Header.Set("user-role", "CLIENT")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

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
