package query

import (
	"context"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
)

// ListBookingsQuery holds the filters for listing bookings.
type ListBookingsQuery struct {
	ClientID    *string
	CompanionID *string
	Status      *string
	Page        int
	PageSize    int
}

// ListBookingsResult contains the paginated result.
type ListBookingsResult struct {
	Bookings []*aggregate.Booking
	Total    int64
	Page     int
	PageSize int
}

// ListBookingsHandler handles the ListBookings query.
type ListBookingsHandler struct {
	repo repository.BookingRepository
}

func NewListBookingsHandler(repo repository.BookingRepository) *ListBookingsHandler {
	return &ListBookingsHandler{repo: repo}
}

func (h *ListBookingsHandler) Handle(ctx context.Context, q ListBookingsQuery) (*ListBookingsResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 50 {
		q.PageSize = 20
	}

	filters := repository.BookingFilters{
		ClientID:    q.ClientID,
		CompanionID: q.CompanionID,
		Status:      q.Status,
		Page:        q.Page,
		PageSize:    q.PageSize,
	}

	bookings, total, err := h.repo.FindByFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &ListBookingsResult{
		Bookings: bookings,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}
