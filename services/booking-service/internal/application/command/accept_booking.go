package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// AcceptBookingCmd holds the data for accepting a booking.
type AcceptBookingCmd struct {
	BookingID   string
	CompanionID string
}

// AcceptBookingHandler handles the AcceptBooking command.
type AcceptBookingHandler struct {
	repo repository.BookingRepository
}

func NewAcceptBookingHandler(repo repository.BookingRepository) *AcceptBookingHandler {
	return &AcceptBookingHandler{repo: repo}
}

func (h *AcceptBookingHandler) Handle(ctx context.Context, cmd AcceptBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}
	companionID, err := vo.NewCompanionID(cmd.CompanionID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.Accept(companionID, time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
