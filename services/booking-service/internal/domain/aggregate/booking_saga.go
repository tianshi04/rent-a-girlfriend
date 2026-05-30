package aggregate

import (
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

type BookingAcceptSaga struct {
	ID        string
	BookingID string
	State     vo.SagaState
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewBookingAcceptSaga(id, bookingID string, now time.Time) *BookingAcceptSaga {
	return &BookingAcceptSaga{
		ID:        id,
		BookingID: bookingID,
		State:     vo.SagaStateStarted,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *BookingAcceptSaga) UpdateState(state vo.SagaState, now time.Time) {
	s.State = state
	s.UpdatedAt = now
}
