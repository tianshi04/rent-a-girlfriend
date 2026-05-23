package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

func TestRequestBooking_Success(t *testing.T) {
	repo := NewMockBookingRepository()
	profile := &MockProfileService{}
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	profile.Snapshot = &snap

	finance := &MockFinanceService{}

	handler := command.NewRequestBookingHandler(repo, profile, finance)

	now := time.Now()
	cmd := command.RequestBookingCmd{
		ClientID:    "550e8400-e29b-41d4-a716-446655440001",
		CompanionID: "550e8400-e29b-41d4-a716-446655440002",
		ScenarioID:  "scenario-123",
		StartTime:   now.Add(4 * time.Hour), // INV-B01: >3h advance (using 4h to avoid time-race)
	}

	booking, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if booking == nil {
		t.Fatal("expected booking to be created, got nil")
	}

	if booking.Status() != vo.StatusPending {
		t.Errorf("expected status PENDING, got %s", booking.Status())
	}

	if len(finance.FrozenCalls) != 1 || finance.FrozenCalls[0] != cmd.ClientID {
		t.Errorf("expected FreezeCoin to be called once for %s, got %v", cmd.ClientID, finance.FrozenCalls)
	}

	// Verify it was persisted
	saved, err := repo.FindByID(context.Background(), booking.ID())
	if err != nil {
		t.Fatalf("failed to retrieve saved booking: %v", err)
	}
	if saved.ID() != booking.ID() {
		t.Error("saved booking ID mismatch")
	}
}

func TestRequestBooking_INV_B04_CompanionCapExceeded(t *testing.T) {
	repo := NewMockBookingRepository()
	profile := &MockProfileService{}
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	profile.Snapshot = &snap

	finance := &MockFinanceService{}

	handler := command.NewRequestBookingHandler(repo, profile, finance)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")

	now := time.Now()
	// Pre-fill 10 pending bookings to reach cap limit
	for i := 0; i < 10; i++ {
		bid := vo.NewBookingID()
		tr, _ := vo.NewTimeRange(now.Add(time.Duration(100+i*5)*time.Hour), now.Add(time.Duration(102+i*5)*time.Hour))
		b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
		_ = repo.Save(context.Background(), b)
	}

	cmd := command.RequestBookingCmd{
		ClientID:    clientID.String(),
		CompanionID: companionID.String(),
		ScenarioID:  "scenario-123",
		StartTime:   now.Add(4 * time.Hour),
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrPendingCapExceeded) {
		t.Errorf("expected ErrPendingCapExceeded, got %v", err)
	}
}

func TestRequestBooking_ClientOverlap(t *testing.T) {
	repo := NewMockBookingRepository()
	profile := &MockProfileService{}
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	profile.Snapshot = &snap
	finance := &MockFinanceService{}

	handler := command.NewRequestBookingHandler(repo, profile, finance)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")

	now := time.Now()
	// Pre-fill an existing booking for the Client in overlapping period [3h, 5h]
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	b1 := aggregate.Reconstitute(vo.NewBookingID(), clientID, companionID, snap, tr, vo.StatusAccepted, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b1)

	// Try to book in [4h, 6h] -> overlaps!
	cmd := command.RequestBookingCmd{
		ClientID:    clientID.String(),
		CompanionID: companionID.String(),
		ScenarioID:  "scenario-123",
		StartTime:   now.Add(4 * time.Hour),
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrClientBookingOverlap) {
		t.Errorf("expected ErrClientBookingOverlap, got %v", err)
	}
}

func TestRequestBooking_FinanceFreezeFailure_Rollback(t *testing.T) {
	repo := NewMockBookingRepository()
	profile := &MockProfileService{}
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	profile.Snapshot = &snap
	
	// Simulate coin freeze failure
	finance := &MockFinanceService{
		FreezeErr: errors.New("insufficient balance"),
	}

	handler := command.NewRequestBookingHandler(repo, profile, finance)

	now := time.Now()
	cmd := command.RequestBookingCmd{
		ClientID:    "550e8400-e29b-41d4-a716-446655440001",
		CompanionID: "550e8400-e29b-41d4-a716-446655440002",
		ScenarioID:  "scenario-123",
		StartTime:   now.Add(4 * time.Hour),
	}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error from handle, got nil")
	}

	if len(finance.FrozenCalls) != 0 {
		t.Errorf("expected no frozen calls registered due to error, got %v", finance.FrozenCalls)
	}
}
