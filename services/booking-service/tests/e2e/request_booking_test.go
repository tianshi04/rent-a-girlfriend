package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	handler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/handler"
)

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
