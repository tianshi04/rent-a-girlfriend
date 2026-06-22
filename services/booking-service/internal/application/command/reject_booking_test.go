package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

func TestRejectBooking_Success(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewRejectBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	// Must be PENDING to reject
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	cmd := command.RejectBookingCmd{
		BookingID:   bid.String(),
		CompanionID: companionID.String(),
		Reason:      "busy",
	}

	res, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Status() != vo.StatusRejected {
		t.Errorf("expected REJECTED status, got %s", res.Status())
	}

	// Verify CoinsUnfreezeRequested event is triggered
	events := res.Events()
	var hasUnfreeze bool
	for _, e := range events {
		if e.EventType() == "finance.coins-unfreeze-requested.v1" {
			hasUnfreeze = true
			if unfreeze, ok := e.(event.CoinsUnfreezeRequested); ok {
				if unfreeze.BookingId != bid.String() {
					t.Errorf("expected booking ID %s, got %s", bid.String(), unfreeze.BookingId)
				}
				if unfreeze.UserId != clientID.String() {
					t.Errorf("expected user ID %s, got %s", clientID.String(), unfreeze.UserId)
				}
				if unfreeze.Amount != price.Amount() {
					t.Errorf("expected amount %d, got %d", price.Amount(), unfreeze.Amount)
				}
			} else {
				t.Errorf("expected event of type CoinsUnfreezeRequested, got %T", e)
			}
		}
	}
	if !hasUnfreeze {
		t.Error("expected CoinsUnfreezeRequested event to be emitted")
	}
}

func TestRejectBooking_UnauthorizedCompanion(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := command.NewRejectBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	// Try with wrong companion ID
	cmd := command.RejectBookingCmd{
		BookingID:   bid.String(),
		CompanionID: "550e8400-e29b-41d4-a716-446655440099",
		Reason:      "busy",
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
