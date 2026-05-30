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
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func TestAcceptBooking_Success(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	handler := command.NewAcceptBookingHandler(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	cmd := command.AcceptBookingCmd{
		BookingID:   bid.String(),
		CompanionID: companionID.String(),
	}

	res, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.ID() != bid {
		t.Errorf("expected booking ID %s, got %s", bid.String(), res.ID().String())
	}

	// Verify saga was created
	saga, err := sagaRepo.FindByBookingID(context.Background(), bid.String())
	if err != nil {
		t.Fatalf("expected saga to be saved, got error: %v", err)
	}
	if saga.State != vo.SagaStateWaitingForEscrow {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateWaitingForEscrow, saga.State)
	}
}

func TestAcceptBooking_UnauthorizedCompanion(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	handler := command.NewAcceptBookingHandler(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	cmd := command.AcceptBookingCmd{
		BookingID:   bid.String(),
		CompanionID: "550e8400-e29b-41d4-a716-446655440099", // different companion
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAcceptBooking_CompanionOverlap(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	handler := command.NewAcceptBookingHandler(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()

	// Existing accepted booking in [3h, 5h]
	tr1, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	b1 := aggregate.Reconstitute(vo.NewBookingID(), clientID, companionID, snap, tr1, vo.StatusAccepted, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b1)

	// New pending booking in [4h, 6h] - overlap!
	tr2, _ := vo.NewTimeRange(now.Add(4*time.Hour), now.Add(6*time.Hour))
	bid2 := vo.NewBookingID()
	b2 := aggregate.Reconstitute(bid2, clientID, companionID, snap, tr2, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b2)

	cmd := command.AcceptBookingCmd{
		BookingID:   bid2.String(),
		CompanionID: companionID.String(),
	}

	_, err := handler.Handle(context.Background(), cmd)
	if !errors.Is(err, domainerr.ErrCompanionBookingOverlap) {
		t.Errorf("expected ErrCompanionBookingOverlap, got %v", err)
	}
}
