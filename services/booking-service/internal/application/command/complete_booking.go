package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// CompleteBookingCmd holds the data for completing a booking.
type CompleteBookingCmd struct {
	BookingID string
	ClientID  string
}

// CompleteBookingHandler handles the CompleteBooking command.
type CompleteBookingHandler struct {
	repo repository.BookingRepository
}

// NewCompleteBookingHandler creates a new CompleteBookingHandler.
func NewCompleteBookingHandler(repo repository.BookingRepository) *CompleteBookingHandler {
	return &CompleteBookingHandler{repo: repo}
}

// Handle processes the CompleteBooking command.
func (h *CompleteBookingHandler) Handle(ctx context.Context, cmd CompleteBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}
	clientID, err := vo.NewClientID(cmd.ClientID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.Complete(clientID, time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}

// SystemCompleteBookingCmd holds the data for system-initiated automated completion.
type SystemCompleteBookingCmd struct {
	BookingID string
}

// SystemCompleteBookingHandler handles the SystemCompleteBooking command.
type SystemCompleteBookingHandler struct {
	repo repository.BookingRepository
}

// NewSystemCompleteBookingHandler creates a new SystemCompleteBookingHandler.
func NewSystemCompleteBookingHandler(repo repository.BookingRepository) *SystemCompleteBookingHandler {
	return &SystemCompleteBookingHandler{repo: repo}
}

// Handle processes the SystemCompleteBooking command.
func (h *SystemCompleteBookingHandler) Handle(ctx context.Context, cmd SystemCompleteBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.SystemComplete(time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
