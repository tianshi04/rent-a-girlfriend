package event

import (
	"time"

	"google.golang.org/protobuf/proto"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
)

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	ToProto() proto.Message
}

// BookingRequested is raised when a Client creates a booking request.
type BookingRequested struct {
	*bookingv1.BookingRequested
}

func (e BookingRequested) EventType() string      { return "booking.booking-requested.v1" }
func (e BookingRequested) OccurredAt() time.Time  { return e.BookingRequested.GetOccurredAt().AsTime() }
func (e BookingRequested) ToProto() proto.Message { return e.BookingRequested }

// BookingAccepted is raised when a Companion accepts a booking.
type BookingAccepted struct {
	*bookingv1.BookingAccepted
}

func (e BookingAccepted) EventType() string      { return "booking.booking-accepted.v1" }
func (e BookingAccepted) OccurredAt() time.Time  { return e.BookingAccepted.GetOccurredAt().AsTime() }
func (e BookingAccepted) ToProto() proto.Message { return e.BookingAccepted }

// BookingRejected is raised when a Companion rejects a booking.
type BookingRejected struct {
	*bookingv1.BookingRejected
}

func (e BookingRejected) EventType() string      { return "booking.booking-rejected.v1" }
func (e BookingRejected) OccurredAt() time.Time  { return e.BookingRejected.GetOccurredAt().AsTime() }
func (e BookingRejected) ToProto() proto.Message { return e.BookingRejected }

// BookingCancelledEarly is raised when booking is cancelled >24h before start.
type BookingCancelledEarly struct {
	*bookingv1.BookingCancelledEarly
}

func (e BookingCancelledEarly) EventType() string {
	return "booking.booking-cancelled-early.v1"
}
func (e BookingCancelledEarly) OccurredAt() time.Time {
	return e.BookingCancelledEarly.GetOccurredAt().AsTime()
}
func (e BookingCancelledEarly) ToProto() proto.Message { return e.BookingCancelledEarly }

// BookingCancelledLate is raised when booking is cancelled <24h before start.
type BookingCancelledLate struct {
	*bookingv1.BookingCancelledLate
}

func (e BookingCancelledLate) EventType() string {
	return "booking.booking-cancelled-late.v1"
}
func (e BookingCancelledLate) OccurredAt() time.Time {
	return e.BookingCancelledLate.GetOccurredAt().AsTime()
}
func (e BookingCancelledLate) ToProto() proto.Message { return e.BookingCancelledLate }

// BookingTimedOut is raised when a PENDING booking exceeds 12h.
type BookingTimedOut struct {
	*bookingv1.BookingTimedOut
}

func (e BookingTimedOut) EventType() string      { return "booking.booking-timed-out.v1" }
func (e BookingTimedOut) OccurredAt() time.Time  { return e.BookingTimedOut.GetOccurredAt().AsTime() }
func (e BookingTimedOut) ToProto() proto.Message { return e.BookingTimedOut }

// BookingCompleted is raised when an ACCEPTED booking completes without dispute.
type BookingCompleted struct {
	*bookingv1.BookingCompleted
}

func (e BookingCompleted) EventType() string      { return "booking.booking-completed.v1" }
func (e BookingCompleted) OccurredAt() time.Time  { return e.BookingCompleted.GetOccurredAt().AsTime() }
func (e BookingCompleted) ToProto() proto.Message { return e.BookingCompleted }

// BookingReserved is raised when a Booking transitions to PENDING status after funds are successfully frozen.
type BookingReserved struct {
	*bookingv1.BookingReserved
}

func (e BookingReserved) EventType() string      { return "booking.booking-reserved.v1" }
func (e BookingReserved) OccurredAt() time.Time  { return e.BookingReserved.GetOccurredAt().AsTime() }
func (e BookingReserved) ToProto() proto.Message { return e.BookingReserved }
