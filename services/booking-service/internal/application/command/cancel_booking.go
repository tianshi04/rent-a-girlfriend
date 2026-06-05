package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// CancelBookingCmd holds the data for cancelling a booking.
type CancelBookingCmd struct {
	BookingID string
	ActorID   string
	ActorRole string // "CLIENT" or "COMPANION" sử dụng enum
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

	// [INV-CANCEL-AUTH] Verify if the actor belongs to the booking they are trying to cancel
	switch actorRole {
	case vo.RoleClient:
		clientID, err := vo.NewClientID(cmd.ActorID)
		if err != nil {
			return nil, err
		}
		if !booking.ClientID().Equals(clientID) {
			return nil, domainerr.ErrUnauthorized
		}
	case vo.RoleCompanion:
		companionID, err := vo.NewCompanionID(cmd.ActorID)
		if err != nil {
			return nil, err
		}
		if !booking.CompanionID().Equals(companionID) {
			return nil, domainerr.ErrUnauthorized
		}
	default:
		return nil, domainerr.ErrUnauthorized
	}

	if err := booking.Cancel(actorRole, time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
