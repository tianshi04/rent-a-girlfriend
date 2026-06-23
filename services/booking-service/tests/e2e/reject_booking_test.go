package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func TestE2E_RejectBooking_Success(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	// Pre-populate a PENDING booking in database
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000123")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000456")
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

	// Call RejectBooking POST /api/v1/bookings/:id/reject
	reqBody := map[string]interface{}{
		"companionId": "00000000-0000-0000-0000-000000000456",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/api/v1/bookings/%s/reject", getBaseURL(), bid.String())
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000456")
	req.Header.Set("user-role", "COMPANION")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for reject booking, got %d", resp.StatusCode)
	}

	// Verify that the booking status was updated to REJECTED in database
	updated, err := repo.FindByID(context.Background(), bid)
	if err != nil {
		t.Fatalf("failed to find booking: %v", err)
	}
	if updated.Status() != vo.StatusRejected {
		t.Errorf("expected booking status to be REJECTED, got %s", updated.Status())
	}
}
