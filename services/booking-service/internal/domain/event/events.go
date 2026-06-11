package event

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// BookingRequested is raised when a Client creates a booking request.
type BookingRequested struct {
	BookingID      string    `json:"bookingId"`
	ClientID       string    `json:"clientId"`
	CompanionID    string    `json:"companionId"`
	Price          int64     `json:"price"`
	StartTime      time.Time `json:"startTime"`
	OccurredAtTime time.Time `json:"occurredAt"`
}

func (e BookingRequested) EventType() string     { return "booking.booking-requested.v1" }
func (e BookingRequested) OccurredAt() time.Time { return e.OccurredAtTime }

// BookingAccepted is raised when a Companion accepts a booking.
type BookingAccepted struct {
	BookingID      string    `json:"bookingId"`
	CompanionID    string    `json:"companionId"`
	OccurredAtTime time.Time `json:"occurredAt"`
}

func (e BookingAccepted) EventType() string     { return "booking.booking-accepted.v1" }
func (e BookingAccepted) OccurredAt() time.Time { return e.OccurredAtTime }

// BookingRejected is raised when a Companion rejects a booking.
type BookingRejected struct {
	BookingID      string    `json:"bookingId"`
	CompanionID    string    `json:"companionId"`
	ClientID       string    `json:"clientId"`
	Reason         string    `json:"reason"`
	OccurredAtTime time.Time `json:"occurredAt"`
}

func (e BookingRejected) EventType() string     { return "booking.booking-rejected.v1" }
func (e BookingRejected) OccurredAt() time.Time { return e.OccurredAtTime }

// BookingCancelledEarly is raised when booking is cancelled >24h before start.
type BookingCancelledEarly struct {
	BookingID      string    `json:"bookingId"`
	ActorID        string    `json:"actorId"`
	ActorRole      string    `json:"actorRole"`
	OccurredAtTime time.Time `json:"occurredAt"`
}

func (e BookingCancelledEarly) EventType() string {
	return "booking.booking-cancelled-early.v1"
}
func (e BookingCancelledEarly) OccurredAt() time.Time { return e.OccurredAtTime }

// BookingCancelledLate is raised when booking is cancelled <24h before start.
type BookingCancelledLate struct {
	BookingID      string    `json:"bookingId"`
	ActorID        string    `json:"actorId"`
	ActorRole      string    `json:"actorRole"`
	OccurredAtTime time.Time `json:"occurredAt"`
}

func (e BookingCancelledLate) EventType() string {
	return "booking.booking-cancelled-late.v1"
}
func (e BookingCancelledLate) OccurredAt() time.Time { return e.OccurredAtTime }

// BookingTimedOut is raised when a PENDING booking exceeds 12h.
type BookingTimedOut struct {
	BookingID      string    `json:"bookingId"`
	ClientID       string    `json:"clientId"`
	OccurredAtTime time.Time `json:"occurredAt"`
}

func (e BookingTimedOut) EventType() string     { return "booking.booking-timed-out.v1" }
func (e BookingTimedOut) OccurredAt() time.Time { return e.OccurredAtTime }

// BookingCompleted is raised when an ACCEPTED booking completes without dispute.
type BookingCompleted struct {
	BookingID      string    `json:"bookingId"`
	CompanionID    string    `json:"companionId"`
	ClientID       string    `json:"clientId"`
	Price          int64     `json:"price"`
	OccurredAtTime time.Time `json:"occurredAt"`
}

func (e BookingCompleted) EventType() string     { return "booking.booking-completed.v1" }
func (e BookingCompleted) OccurredAt() time.Time { return e.OccurredAtTime }
