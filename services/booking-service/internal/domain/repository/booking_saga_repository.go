package repository

import (
	"context"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
)

type BookingSagaRepository interface {
	Save(ctx context.Context, saga *aggregate.BookingAcceptSaga) error
	Update(ctx context.Context, saga *aggregate.BookingAcceptSaga) error
	FindByID(ctx context.Context, id string) (*aggregate.BookingAcceptSaga, error)
	FindByBookingID(ctx context.Context, bookingID string) (*aggregate.BookingAcceptSaga, error)
}
