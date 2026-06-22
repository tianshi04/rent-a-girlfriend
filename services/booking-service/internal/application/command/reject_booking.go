package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// RejectBookingCmd holds the data for rejecting a booking.
type RejectBookingCmd struct {
	BookingID   string
	CompanionID string
	Reason      string
}

// RejectBookingHandler handles the RejectBooking command.
type RejectBookingHandler struct {
	repo repository.BookingRepository
}

func NewRejectBookingHandler(repo repository.BookingRepository) *RejectBookingHandler {
	return &RejectBookingHandler{repo: repo}
}

func (h *RejectBookingHandler) Handle(ctx context.Context, cmd RejectBookingCmd) (*aggregate.Booking, error) {
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

	if err := booking.Reject(companionID, cmd.Reason, time.Now()); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
