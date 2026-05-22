package event

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// BookingRequested is raised when a Client creates a booking request.
type BookingRequested struct {
	BookingID   string    `json:"bookingId"`
	ClientID    string    `json:"clientId"`
	CompanionID string    `json:"companionId"`
	Price       int64     `json:"price"`
	StartTime   time.Time `json:"startTime"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e BookingRequested) EventType() string    { return "com.rentagf.booking.BookingRequested.v1" }
func (e BookingRequested) OccurredAt() time.Time { return e.Timestamp }

// BookingAccepted is raised when a Companion accepts a booking.
type BookingAccepted struct {
	BookingID   string    `json:"bookingId"`
	CompanionID string    `json:"companionId"`
	Price       int64     `json:"price"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e BookingAccepted) EventType() string    { return "com.rentagf.booking.BookingAccepted.v1" }
func (e BookingAccepted) OccurredAt() time.Time { return e.Timestamp }

// BookingRejected is raised when a Companion rejects a booking.
type BookingRejected struct {
	BookingID   string    `json:"bookingId"`
	CompanionID string    `json:"companionId"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e BookingRejected) EventType() string    { return "com.rentagf.booking.BookingRejected.v1" }
func (e BookingRejected) OccurredAt() time.Time { return e.Timestamp }

// BookingCancelledEarly is raised when booking is cancelled >24h before start.
type BookingCancelledEarly struct {
	BookingID string    `json:"bookingId"`
	ActorID   string    `json:"actorId"`
	ActorRole string    `json:"actorRole"`
	Timestamp time.Time `json:"timestamp"`
}

func (e BookingCancelledEarly) EventType() string    { return "com.rentagf.booking.BookingCancelledEarly.v1" }
func (e BookingCancelledEarly) OccurredAt() time.Time { return e.Timestamp }

// BookingCancelledLate is raised when booking is cancelled <24h before start.
type BookingCancelledLate struct {
	BookingID string    `json:"bookingId"`
	ActorID   string    `json:"actorId"`
	ActorRole string    `json:"actorRole"`
	Timestamp time.Time `json:"timestamp"`
}

func (e BookingCancelledLate) EventType() string    { return "com.rentagf.booking.BookingCancelledLate.v1" }
func (e BookingCancelledLate) OccurredAt() time.Time { return e.Timestamp }

// BookingTimedOut is raised when a PENDING booking exceeds 12h.
type BookingTimedOut struct {
	BookingID string    `json:"bookingId"`
	ClientID  string    `json:"clientId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e BookingTimedOut) EventType() string    { return "com.rentagf.booking.BookingTimedOut.v1" }
func (e BookingTimedOut) OccurredAt() time.Time { return e.Timestamp }

// BookingCompleted is raised when an ACCEPTED booking completes without dispute.
type BookingCompleted struct {
	BookingID   string    `json:"bookingId"`
	CompanionID string    `json:"companionId"`
	ClientID    string    `json:"clientId"`
	Price       int64     `json:"price"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e BookingCompleted) EventType() string    { return "com.rentagf.booking.BookingCompleted.v1" }
func (e BookingCompleted) OccurredAt() time.Time { return e.Timestamp }
