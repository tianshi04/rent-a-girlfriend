package vo

import (
	"fmt"

	"github.com/google/uuid"
)

// BookingID is a unique identifier for a Booking aggregate.
type BookingID struct {
	value uuid.UUID
}

// NewBookingID generates a new random BookingID.
func NewBookingID() BookingID {
	return BookingID{value: uuid.New()}
}

// BookingIDFromString parses a string into a BookingID.
func BookingIDFromString(s string) (BookingID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return BookingID{}, fmt.Errorf("invalid booking id: %w", err)
	}
	return BookingID{value: id}, nil
}

// String returns the string representation.
func (b BookingID) String() string {
	return b.value.String()
}

// IsZero checks if the BookingID is empty.
func (b BookingID) IsZero() bool {
	return b.value == uuid.Nil
}

// Equals checks equality with another BookingID.
func (b BookingID) Equals(other BookingID) bool {
	return b.value == other.value
}
