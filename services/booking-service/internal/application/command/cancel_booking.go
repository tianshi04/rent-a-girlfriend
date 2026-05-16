package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// CancelBookingCmd holds the data for cancelling a booking.
type CancelBookingCmd struct {
	BookingID string
	ActorID   string
	ActorRole string // "CLIENT" or "COMPANION"
}

// CancelBookingHandler handles the CancelBooking command.
type CancelBookingHandler struct {
	repo repository.BookingRepository
}

func NewCancelBookingHandler(repo repository.BookingRepository) *CancelBookingHandler {
	return &CancelBookingHandler{repo: repo}
}

func (h *CancelBookingHandler) Handle(ctx context.Context, cmd CancelBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}
	actorRole, err := vo.NewActorRole(cmd.ActorRole)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.Cancel(actorRole, time.Now()); err != nil {
		return nil, err
	}

	// Note: Refund/penalty logic is handled by Finance Service via events (Phase 2).
	// The domain event (BookingCancelledEarly/Late) carries isLate info for Finance to decide.

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
