package query

import (
	"context"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// GetBookingQuery holds the data for fetching a single booking.
type GetBookingQuery struct {
	BookingID string
}

// GetBookingHandler handles the GetBooking query.
type GetBookingHandler struct {
	repo repository.BookingRepository
}

func NewGetBookingHandler(repo repository.BookingRepository) *GetBookingHandler {
	return &GetBookingHandler{repo: repo}
}

func (h *GetBookingHandler) Handle(ctx context.Context, q GetBookingQuery) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(q.BookingID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, domainerr.ErrBookingNotFound
	}

	return booking, nil
}
