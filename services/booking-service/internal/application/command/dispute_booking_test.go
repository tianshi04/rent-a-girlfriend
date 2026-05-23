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

func TestDisputeBooking_Success(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewDisputeBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	
	bid := vo.NewBookingID()
	// Must be ACCEPTED (or similar) to dispute
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusAccepted, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	cmd := command.DisputeBookingCmd{
		BookingID: bid.String(),
	}

	res, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Status() != vo.StatusDisputed {
		t.Errorf("expected DISPUTED status, got %s", res.Status())
	}
}

func TestDisputeBooking_InvalidState(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewDisputeBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	
	bid := vo.NewBookingID()
	// PENDING status cannot be disputed directly
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	cmd := command.DisputeBookingCmd{
		BookingID: bid.String(),
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}
