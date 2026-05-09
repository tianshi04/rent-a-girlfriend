package vo

import (
	"regexp"
	"strings"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email is a Value Object that validates and encapsulates an email address.
type Email struct {
	value string
}

// NewEmail creates a validated Email VO.
func NewEmail(address string) (Email, error) {
	normalized := strings.TrimSpace(strings.ToLower(address))
	if normalized == "" || !emailRegex.MatchString(normalized) {
		return Email{}, domainerr.ErrInvalidEmail
	}
	return Email{value: normalized}, nil
}

// String returns the email string.
func (e Email) String() string {
	return e.value
}

// Equals checks equality with another Email.
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}
