package vo_test

import (
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

func TestBookingID_NewAndParse(t *testing.T) {
	id := vo.NewBookingID()
	if id.IsZero() {
		t.Error("new BookingID should not be zero")
	}

	parsed, err := vo.BookingIDFromString(id.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !id.Equals(parsed) {
		t.Error("parsed ID should equal original")
	}
}

func TestBookingID_InvalidString(t *testing.T) {
	_, err := vo.BookingIDFromString("not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID string")
	}
}

func TestMoney_Validation(t *testing.T) {
	_, err := vo.NewMoney(-1)
	if err == nil {
		t.Error("expected error for negative money")
	}

	m, err := vo.NewMoney(500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Amount() != 500 {
		t.Errorf("expected 500, got %d", m.Amount())
	}
}

func TestTimeRange_Validation(t *testing.T) {
	now := time.Now()
	start := now.Add(3 * time.Hour)
	end := start.Add(2 * time.Hour)

	tr, err := vo.NewTimeRange(start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// INV-B01: advance booking check
	if err := tr.ValidateAdvanceBooking(now); err != nil {
		t.Errorf("should pass advance booking check: %v", err)
	}

	// Fail: start too soon
	tooSoon, _ := vo.NewTimeRange(now.Add(30*time.Minute), now.Add(150*time.Minute))
	if err := tooSoon.ValidateAdvanceBooking(now); err == nil {
		t.Error("expected error for too soon booking")
	}

	// INV-B02: duration check
	if err := tr.ValidateDuration(120); err != nil {
		t.Errorf("should pass duration check: %v", err)
	}
	if err := tr.ValidateDuration(60); err == nil {
		t.Error("expected error for mismatched duration")
	}
}

func TestTimeRange_EndBeforeStart(t *testing.T) {
	now := time.Now()
	_, err := vo.NewTimeRange(now.Add(5*time.Hour), now.Add(3*time.Hour))
	if err == nil {
		t.Error("expected error when end is before start")
	}
}

func TestScenarioSnapshot_Validation(t *testing.T) {
	_, err := vo.NewScenarioSnapshot(vo.MustMoney(0), 120)
	if err == nil {
		t.Error("expected error for zero price")
	}

	_, err = vo.NewScenarioSnapshot(vo.MustMoney(500), 0)
	if err == nil {
		t.Error("expected error for zero duration")
	}

	s, err := vo.NewScenarioSnapshot(vo.MustMoney(500), 120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Price().Amount() != 500 || s.DurationMinutes() != 120 {
		t.Error("snapshot fields mismatch")
	}
}

func TestBookingStatus_Transitions(t *testing.T) {
	if !vo.StatusPending.CanAccept() {
		t.Error("PENDING should allow accept")
	}
	if !vo.StatusPending.CanReject() {
		t.Error("PENDING should allow reject")
	}
	if vo.StatusAccepted.CanAccept() {
		t.Error("ACCEPTED should not allow accept")
	}
	if vo.StatusCompleted.CanCancel() {
		t.Error("COMPLETED should not allow cancel")
	}
	if vo.StatusCancelled.CanCancel() {
		t.Error("CANCELLED should not allow cancel")
	}
	if !vo.StatusAccepted.CanCancel() {
		t.Error("ACCEPTED should allow cancel")
	}
}

func TestActorRole_Validation(t *testing.T) {
	_, err := vo.NewActorRole("CLIENT")
	if err != nil {
		t.Error("CLIENT should be valid")
	}
	_, err = vo.NewActorRole("COMPANION")
	if err != nil {
		t.Error("COMPANION should be valid")
	}
	_, err = vo.NewActorRole("SYSTEM")
	if err != nil {
		t.Error("SYSTEM should be valid")
	}
	_, err = vo.NewActorRole("ADMIN")
	if err == nil {
		t.Error("ADMIN should be invalid actor role")
	}
}

func TestIsLateCancel(t *testing.T) {
	now := time.Now()

	// Start in 3h → within 24h → late
	start := now.Add(3 * time.Hour)
	end := start.Add(2 * time.Hour)
	tr, _ := vo.NewTimeRange(start, end)
	if !tr.IsLateCancel(now) {
		t.Error("should be late cancel when start is within 24h")
	}

	// Start in 48h → beyond 24h → early
	start2 := now.Add(48 * time.Hour)
	end2 := start2.Add(2 * time.Hour)
	tr2, _ := vo.NewTimeRange(start2, end2)
	if tr2.IsLateCancel(now) {
		t.Error("should be early cancel when start is beyond 24h")
	}
}
