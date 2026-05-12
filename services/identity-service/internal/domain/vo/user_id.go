package vo

import (
	"github.com/google/uuid"
)

// UserID is a Value Object wrapping a UUID for user identification.
type UserID struct {
	value uuid.UUID
}

// NewUserID generates a new random UserID.
func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

// ParseUserID parses a string into a UserID.
func ParseUserID(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UserID{}, err
	}
	return UserID{value: id}, nil
}

// String returns the string representation.
func (u UserID) String() string {
	return u.value.String()
}

// UUID returns the underlying uuid.UUID.
func (u UserID) UUID() uuid.UUID {
	return u.value
}

// IsZero checks if the UserID is empty.
func (u UserID) IsZero() bool {
	return u.value == uuid.Nil
}

// Equals checks equality with another UserID.
func (u UserID) Equals(other UserID) bool {
	return u.value == other.value
}
