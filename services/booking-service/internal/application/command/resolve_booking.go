package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// ResolveBookingCmd holds the data for resolving a booking.
type ResolveBookingCmd struct {
	BookingID string
}

// ResolveBookingHandler handles the ResolveBooking command.
type ResolveBookingHandler struct {
	repo repository.BookingRepository
}

// NewResolveBookingHandler creates a new ResolveBookingHandler instance.
func NewResolveBookingHandler(repo repository.BookingRepository) *ResolveBookingHandler {
	return &ResolveBookingHandler{repo: repo}
}

// Handle processes the ResolveBooking command, changing status to RESOLVED.
func (h *ResolveBookingHandler) Handle(ctx context.Context, cmd ResolveBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.Resolve(time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
