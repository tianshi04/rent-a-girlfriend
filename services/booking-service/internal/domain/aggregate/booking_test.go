package aggregate_test

import (
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

func validBookingParams(t *testing.T) (vo.ClientID, vo.CompanionID, vo.ScenarioSnapshot, vo.TimeRange, time.Time) {
	t.Helper()
	now := time.Now()
	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	scenario, _ := vo.NewScenarioSnapshot(price, 120) // 120 minutes
	start := now.Add(3 * time.Hour)
	end := start.Add(120 * time.Minute)
	timeRange, _ := vo.NewTimeRange(start, end)
	return clientID, companionID, scenario, timeRange, now
}

func TestNewBooking_Success(t *testing.T) {
	clientID, companionID, scenario, timeRange, now := validBookingParams(t)

	booking, err := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.Status() != vo.StatusPending {
		t.Errorf("expected status PENDING, got %s", booking.Status())
	}
	if booking.ID().IsZero() {
		t.Error("expected non-zero booking ID")
	}
	events := booking.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType() != "com.rentagf.booking.BookingRequested.v1" {
		t.Errorf("expected BookingRequested event, got %s", events[0].EventType())
	}
}

func TestNewBooking_INV_B01_TooSoon(t *testing.T) {
	now := time.Now()
	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	scenario, _ := vo.NewScenarioSnapshot(price, 120)
	// Start only 1 hour from now (violates INV-B01: need >= 2h)
	start := now.Add(1 * time.Hour)
	end := start.Add(120 * time.Minute)
	timeRange, _ := vo.NewTimeRange(start, end)

	_, err := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	if err == nil {
		t.Fatal("expected error for booking too soon")
	}
}

func TestNewBooking_INV_B02_DurationMismatch(t *testing.T) {
	now := time.Now()
	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	scenario, _ := vo.NewScenarioSnapshot(price, 120) // 120 min scenario
	start := now.Add(3 * time.Hour)
	end := start.Add(60 * time.Minute) // only 60 min (mismatch)
	timeRange, _ := vo.NewTimeRange(start, end)

	_, err := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	if err == nil {
		t.Fatal("expected error for duration mismatch")
	}
}

func TestAccept_INV_B03_Success(t *testing.T) {
	clientID, companionID, scenario, timeRange, now := validBookingParams(t)
	booking, _ := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	_ = booking.Events() // clear

	err := booking.Accept(companionID, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.Status() != vo.StatusAccepted {
		t.Errorf("expected ACCEPTED, got %s", booking.Status())
	}
}

func TestAccept_INV_B03_WrongStatus(t *testing.T) {
	clientID, companionID, scenario, timeRange, now := validBookingParams(t)
	booking, _ := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	_ = booking.Accept(companionID, now) // PENDING -> ACCEPTED

	// Try to accept again (now ACCEPTED, should fail)
	err := booking.Accept(companionID, now)
	if err == nil {
		t.Fatal("expected error for invalid status transition")
	}
}

func TestAccept_WrongCompanion(t *testing.T) {
	clientID, companionID, scenario, timeRange, now := validBookingParams(t)
	booking, _ := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)

	otherCompanion, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440099")
	err := booking.Accept(otherCompanion, now)
	if err == nil {
		t.Fatal("expected unauthorized error for wrong companion")
	}
}

func TestReject_INV_B03_Success(t *testing.T) {
	clientID, companionID, scenario, timeRange, now := validBookingParams(t)
	booking, _ := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	_ = booking.Events()

	err := booking.Reject(companionID, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.Status() != vo.StatusRejected {
		t.Errorf("expected REJECTED, got %s", booking.Status())
	}
}

func TestCancel_INV_B05_Success(t *testing.T) {
	clientID, companionID, scenario, timeRange, now := validBookingParams(t)
	booking, _ := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	_ = booking.Events()

	err := booking.Cancel(vo.RoleClient, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.Status() != vo.StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", booking.Status())
	}
}

func TestCancel_INV_B05_AlreadyCompleted(t *testing.T) {
	// Reconstitute a COMPLETED booking
	bookingID := vo.NewBookingID()
	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	scenario, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	start := now.Add(3 * time.Hour)
	end := start.Add(120 * time.Minute)
	timeRange, _ := vo.NewTimeRange(start, end)

	booking := aggregate.Reconstitute(
		bookingID, clientID, companionID, scenario, timeRange,
		vo.StatusCompleted, "", false, 1, now, now,
	)

	err := booking.Cancel(vo.RoleClient, now)
	if err == nil {
		t.Fatal("expected error for cancelling completed booking")
	}
}

func TestCancel_LateCancellation(t *testing.T) {
	now := time.Now()
	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	scenario, _ := vo.NewScenarioSnapshot(price, 120)
	// Start in 3h (within 24h threshold → late cancel)
	start := now.Add(3 * time.Hour)
	end := start.Add(120 * time.Minute)
	timeRange, _ := vo.NewTimeRange(start, end)

	booking, _ := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	_ = booking.Events()

	_ = booking.Cancel(vo.RoleClient, now)
	if !booking.IsLateCancel() {
		t.Error("expected late cancellation to be true")
	}

	events := booking.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType() != "com.rentagf.booking.BookingCancelledLate.v1" {
		t.Errorf("expected BookingCancelledLate, got %s", events[0].EventType())
	}
}

func TestCancel_EarlyCancellation(t *testing.T) {
	now := time.Now()
	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	scenario, _ := vo.NewScenarioSnapshot(price, 120)
	// Start in 48h (> 24h threshold → early cancel)
	start := now.Add(48 * time.Hour)
	end := start.Add(120 * time.Minute)
	timeRange, _ := vo.NewTimeRange(start, end)

	booking, _ := aggregate.NewBooking(clientID, companionID, scenario, timeRange, now)
	_ = booking.Events()

	_ = booking.Cancel(vo.RoleClient, now)
	if booking.IsLateCancel() {
		t.Error("expected late cancellation to be false")
	}

	events := booking.Events()
	if events[0].EventType() != "com.rentagf.booking.BookingCancelledEarly.v1" {
		t.Errorf("expected BookingCancelledEarly, got %s", events[0].EventType())
	}
}
