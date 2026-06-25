package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func TestE2E_GetBooking_SecurityMatrix(t *testing.T) {
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

	// 1. Test Client access to own booking -> OK (200)
	url := fmt.Sprintf("%s/api/v1/bookings/%s", getBaseURL(), bid.String())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000111")
	req.Header.Set("user-role", "CLIENT")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for owner client, got %d", resp.StatusCode)
	}

	// 2. Test Other Client access to booking -> Forbidden (403)
	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000999")
	req.Header.Set("user-role", "CLIENT")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for different client, got %d", resp.StatusCode)
	}

	// 3. Test Admin access to booking -> OK (200)
	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("user-role", "ADMIN")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for admin, got %d", resp.StatusCode)
	}
}

// TestE2E_GetBooking_Companion_HidesPendingReserving asserts that a companion
// cannot view details of a booking in PENDING_RESERVING status.
func TestE2E_GetBooking_Companion_HidesPendingReserving(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000111")
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000222")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPendingReserving, "", false, 1, now, now)

	err := repo.Save(context.Background(), b)
	if err != nil {
		t.Fatalf("failed to save booking: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/bookings/%s", getBaseURL(), bid.String())

	// Companion accesses -> Forbidden (403)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000222")
	req.Header.Set("user-role", "COMPANION")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for companion on PENDING_RESERVING, got %d", resp.StatusCode)
	}
}

// TestE2E_GetBooking_Companion_AsClient_SeesPendingReserving asserts that a user with
// COMPANION role who acts as the CLIENT of the booking CAN view its details in PENDING_RESERVING.
func TestE2E_GetBooking_Companion_AsClient_SeesPendingReserving(t *testing.T) {
	db := getTestDB(t)
	truncateTables(db)
	defer truncateTables(db)

	repo := persistence.NewBookingRepository(db)

	clientID, _ := vo.NewClientID("00000000-0000-0000-0000-000000000222") // User ID (acting as client)
	companionID, _ := vo.NewCompanionID("00000000-0000-0000-0000-000000000333")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPendingReserving, "", false, 1, now, now)

	err := repo.Save(context.Background(), b)
	if err != nil {
		t.Fatalf("failed to save booking: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/bookings/%s", getBaseURL(), bid.String())

	// Caller is COMPANION, but is the client of the booking -> OK (200)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-id", "00000000-0000-0000-0000-000000000222")
	req.Header.Set("user-role", "COMPANION")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for companion acting as client on PENDING_RESERVING, got %d", resp.StatusCode)
	}
}
