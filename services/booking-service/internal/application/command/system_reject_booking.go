package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

type SystemRejectBookingCmd struct {
	BookingID string
}

type SystemRejectBookingHandler struct {
	repo repository.BookingRepository
}

func NewSystemRejectBookingHandler(repo repository.BookingRepository) *SystemRejectBookingHandler {
	return &SystemRejectBookingHandler{
		repo: repo,
	}
}

func (h *SystemRejectBookingHandler) Handle(ctx context.Context, cmd SystemRejectBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.SystemTimeout(time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
