package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// DisputeBookingCmd holds the data for disputing a booking.
type DisputeBookingCmd struct {
	BookingID string
}

// DisputeBookingHandler handles the DisputeBooking command.
type DisputeBookingHandler struct {
	repo repository.BookingRepository
}

// NewDisputeBookingHandler creates a new DisputeBookingHandler.
func NewDisputeBookingHandler(repo repository.BookingRepository) *DisputeBookingHandler {
	return &DisputeBookingHandler{repo: repo}
}

// Handle processes the DisputeBooking command.
func (h *DisputeBookingHandler) Handle(ctx context.Context, cmd DisputeBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.Dispute(time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
