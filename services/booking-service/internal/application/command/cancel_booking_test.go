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

func TestCancelBooking_SuccessClientEarly(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewCancelBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	// Start in 48h (early cancellation)
	tr, _ := vo.NewTimeRange(now.Add(48*time.Hour), now.Add(50*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusAccepted, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	cmd := command.CancelBookingCmd{
		BookingID: bid.String(),
		ActorID:   clientID.String(),
		ActorRole: "CLIENT",
	}

	res, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Status() != vo.StatusCancelled {
		t.Errorf("expected CANCELLED status, got %s", res.Status())
	}
	if res.IsLateCancel() {
		t.Error("expected early cancellation to have IsLateCancel false")
	}
}

func TestCancelBooking_SuccessClientLate(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewCancelBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	// Start in 3h (late cancellation)
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusAccepted, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	cmd := command.CancelBookingCmd{
		BookingID: bid.String(),
		ActorID:   clientID.String(),
		ActorRole: "CLIENT",
	}

	res, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Status() != vo.StatusCancelled {
		t.Errorf("expected CANCELLED status, got %s", res.Status())
	}
	if !res.IsLateCancel() {
		t.Error("expected late cancellation to have IsLateCancel true")
	}
}

func TestCancelBooking_UnauthorizedClient(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewCancelBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(48*time.Hour), now.Add(50*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusAccepted, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	// Try to cancel with random other client ID
	cmd := command.CancelBookingCmd{
		BookingID: bid.String(),
		ActorID:   "550e8400-e29b-41d4-a716-446655440099",
		ActorRole: "CLIENT",
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestCancelBooking_InvalidState(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewCancelBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(48*time.Hour), now.Add(50*time.Hour))

	bid := vo.NewBookingID()
	// Reconstitute COMPLETED booking (should fail to cancel)
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusCompleted, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	cmd := command.CancelBookingCmd{
		BookingID: bid.String(),
		ActorID:   clientID.String(),
		ActorRole: "CLIENT",
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}
