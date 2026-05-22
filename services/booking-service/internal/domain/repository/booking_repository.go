package repository

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// BookingRepository is the port interface for persisting Booking aggregates.
// Implementations reside in the infrastructure layer.
type BookingRepository interface {
	// Save persists a new Booking aggregate.
	Save(ctx context.Context, booking *aggregate.Booking) error

	// Update persists changes to an existing Booking with optimistic locking.
	Update(ctx context.Context, booking *aggregate.Booking) error

	// FindByID retrieves a Booking by its ID.
	FindByID(ctx context.Context, id vo.BookingID) (*aggregate.Booking, error)

	// CountPendingByCompanion returns the number of PENDING bookings
	// for a specific companion. Used for INV-B04 validation.
	CountPendingByCompanion(ctx context.Context, companionID vo.CompanionID) (int64, error)

	// HasOverlappingBooking checks if a client or companion has any bookings in the given statuses overlapping with [startTime, endTime].
	HasOverlappingBooking(ctx context.Context, actorID string, isCompanion bool, statuses []vo.BookingStatus, startTime, endTime time.Time) (bool, error)

	// FindByFilters retrieves bookings with optional filtering and pagination.
	FindByFilters(ctx context.Context, filters BookingFilters) ([]*aggregate.Booking, int64, error)

	// FindAcceptedBookingsPastEndTimeBuffer retrieves ACCEPTED bookings past end_time + buffer.
	FindAcceptedBookingsPastEndTimeBuffer(ctx context.Context, now time.Time, buffer time.Duration) ([]*aggregate.Booking, error)
}

// BookingFilters contains optional filter criteria for listing bookings.
type BookingFilters struct {
	ClientID    *string
	CompanionID *string
	Statuses    []string
	Page        int64
	PageSize    int64
}
