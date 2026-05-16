package repository

import (
	"context"

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

	// CountPendingByClientAndCompanion returns the number of PENDING bookings
	// for a specific client-companion pair. Used for INV-B04 validation.
	CountPendingByClientAndCompanion(ctx context.Context, clientID vo.ClientID, companionID vo.CompanionID) (int, error)

	// FindByFilters retrieves bookings with optional filtering and pagination.
	FindByFilters(ctx context.Context, filters BookingFilters) ([]*aggregate.Booking, int64, error)
}

// BookingFilters contains optional filter criteria for listing bookings.
type BookingFilters struct {
	ClientID    *string
	CompanionID *string
	Status      *string
	Page        int
	PageSize    int
}
