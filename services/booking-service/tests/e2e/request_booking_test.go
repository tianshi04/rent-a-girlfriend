package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func TestE2E_RequestBooking_SuccessAndFailure(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	// --- 1. SUCCESS PATH ---
	reqBody := map[string]interface{}{
		"clientId":    "00000000-0000-0000-0000-000000000123",
		"companionId": "00000000-0000-0000-0000-000000000456",
		"scenarioId":  "00000000-0000-0000-0000-000000000789",
		"startTime":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/api/v1/bookings", getBaseURL())
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000123")
	req.Header.Set("user-role", "CLIENT")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for request booking, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var successResp struct {
		BookingId string `json:"bookingId"`
		Status    string `json:"status"`
		Message   string `json:"message"`
	}
	_ = json.Unmarshal(body, &successResp)

	if successResp.BookingId == "" {
		t.Error("expected a non-empty booking ID")
	}
	if successResp.Status != "BOOKING_STATUS_PENDING_RESERVING" {
		t.Errorf("expected booking status BOOKING_STATUS_PENDING_RESERVING, got %s", successResp.Status)
	}

	// Verify that the booking is saved in repository
	bid, err := vo.BookingIDFromString(successResp.BookingId)
	if err != nil {
		t.Fatalf("invalid booking ID in response: %v", err)
	}
	bookingInDB, err := repo.FindByID(context.Background(), bid)
	if err != nil {
		t.Fatalf("booking was not saved in repository: %v", err)
	}
	if bookingInDB.Status() != vo.StatusPendingReserving {
		t.Errorf("expected booking status in DB to be PENDING_RESERVING, got %s", bookingInDB.Status())
	}

	// --- 2. FAILURE PATH: Insufficient Funds ---
	// "expensive" scenario triggers price = 5000 in mock profile service.
	// Since coin freeze is now asynchronous, it will succeed at the HTTP level with 200 OK and status PENDING_RESERVING.
	reqBodyFailure := map[string]interface{}{
		"clientId":    "00000000-0000-0000-0000-000000000123",
		"companionId": "00000000-0000-0000-0000-000000000456",
		"scenarioId":  "expensive",
		"startTime":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}
	bodyBytesFailure, _ := json.Marshal(reqBodyFailure)

	reqFail, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytesFailure))
	reqFail.Header.Set("Content-Type", "application/json")
	reqFail.Header.Set("user-id", "00000000-0000-0000-0000-000000000123")
	reqFail.Header.Set("user-role", "CLIENT")

	respFail, err := http.DefaultClient.Do(reqFail)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	defer func() { _ = respFail.Body.Close() }()

	if respFail.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for async booking request, got %d", respFail.StatusCode)
	}
}
