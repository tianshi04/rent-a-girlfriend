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

// TestE2E_ListBookings_Companion_HidesPendingReserving asserts that a companion
// cannot see bookings in PENDING_RESERVING status, since the coin reservation
// SAGA is still in-flight and the companion has no actionable role at this stage.
func TestE2E_ListBookings_Companion_HidesPendingReserving(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000333")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000444")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()

	// Seed a booking with PENDING_RESERVING status
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPendingReserving, "", false, 1, now, now)
	if err := repo.Save(context.Background(), b); err != nil {
		t.Fatalf("failed to save booking: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/bookings", getBaseURL())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000444")
	req.Header.Set("user-role", "COMPANION")

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

	if len(listResp.Bookings) != 0 {
		t.Errorf("companion should not see PENDING_RESERVING bookings, got %+v", listResp)
	}
}

// TestE2E_ListBookings_Companion_AsClient_SeesPendingReserving asserts that if a user
// with COMPANION role acts as the CLIENT in a booking, they CAN see their own booking
// even if it is in PENDING_RESERVING status.
func TestE2E_ListBookings_Companion_AsClient_SeesPendingReserving(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	// Here clientID belongs to the companion who is calling the API
	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000444")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000555")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()

	// Seed a booking in PENDING_RESERVING status
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPendingReserving, "", false, 1, now, now)
	if err := repo.Save(context.Background(), b); err != nil {
		t.Fatalf("failed to save booking: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/bookings?view=pending", getBaseURL())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000444")
	req.Header.Set("user-role", "COMPANION") // Caller is COMPANION, but client of this booking

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
		t.Errorf("companion acting as client should see their own PENDING_RESERVING booking, got %+v", listResp)
	}
}

// TestE2E_ListBookings_Admin_SeesAll asserts that an admin sees all bookings,
// including PENDING_RESERVING status, with no user filter.
func TestE2E_ListBookings_Admin_SeesAll(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000333")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000444")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()

	// Seed a booking with PENDING_RESERVING status
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPendingReserving, "", false, 1, now, now)
	if err := repo.Save(context.Background(), b); err != nil {
		t.Fatalf("failed to save booking: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/bookings", getBaseURL())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "admin-id")
	req.Header.Set("user-role", "ADMIN")

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
		t.Errorf("admin should see all bookings, got %+v", listResp)
	}
}

