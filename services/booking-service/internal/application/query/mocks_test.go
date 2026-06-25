package query_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// MockBookingRepository implements repository.BookingRepository in-memory for query testing.
type MockBookingRepository struct {
	mu        sync.RWMutex
	bookings  map[string]*aggregate.Booking
	SaveError error
	FindError error
}

func NewMockBookingRepository() *MockBookingRepository {
	return &MockBookingRepository{
		bookings: make(map[string]*aggregate.Booking),
	}
}

func (m *MockBookingRepository) Save(ctx context.Context, booking *aggregate.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SaveError != nil {
		return m.SaveError
	}
	m.bookings[booking.ID().String()] = booking
	return nil
}

func (m *MockBookingRepository) Update(ctx context.Context, booking *aggregate.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bookings[booking.ID().String()] = booking
	return nil
}

func (m *MockBookingRepository) FindByID(ctx context.Context, id vo.BookingID) (*aggregate.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.FindError != nil {
		return nil, m.FindError
	}
	b, ok := m.bookings[id.String()]
	if !ok {
		return nil, errors.New("booking not found")
	}
	return b, nil
}

func (m *MockBookingRepository) CountPendingByCompanion(ctx context.Context, companionID vo.CompanionID) (int64, error) {
	return 0, nil
}

func (m *MockBookingRepository) HasOverlappingBooking(ctx context.Context, actorID string, isCompanion bool, statuses []vo.BookingStatus, startTime, endTime time.Time) (bool, error) {
	return false, nil
}

func (m *MockBookingRepository) FindByFilters(ctx context.Context, filters repository.BookingFilters) ([]*aggregate.Booking, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*aggregate.Booking
	for _, b := range m.bookings {
		if filters.UserID != nil {
			if b.ClientID().String() != *filters.UserID && b.CompanionID().String() != *filters.UserID {
				continue
			}
		}
		if len(filters.Statuses) > 0 {
			matchStatus := false
			for _, st := range filters.Statuses {
				if string(b.Status()) == st {
					matchStatus = true
					break
				}
			}
			if !matchStatus {
				continue
			}
		}
		result = append(result, b)
	}

	// Apply fake pagination
	start := int((filters.Page - 1) * filters.PageSize)
	if start > len(result) {
		return []*aggregate.Booking{}, int64(len(result)), nil
	}
	end := start + int(filters.PageSize)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], int64(len(result)), nil
}

func (m *MockBookingRepository) FindAcceptedBookingsPastEndTimeBuffer(ctx context.Context, now time.Time, buffer time.Duration) ([]*aggregate.Booking, error) {
	return nil, nil
}

func (m *MockBookingRepository) FindPendingBookingsEligibleForTimeout(ctx context.Context, now time.Time) ([]*aggregate.Booking, error) {
	return nil, nil
}
