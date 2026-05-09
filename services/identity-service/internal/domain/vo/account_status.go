package vo

// AccountStatus represents the activation state of a user account.
type AccountStatus string

const (
	StatusActive AccountStatus = "ACTIVE"
	StatusLocked AccountStatus = "LOCKED"
)

// CanLogin returns true if the account is allowed to authenticate.
// Enforces [INV-ID01].
func (s AccountStatus) CanLogin() bool {
	return s == StatusActive
}

// CanLock returns true if the account can transition to LOCKED.
func (s AccountStatus) CanLock() bool {
	return s == StatusActive
}

// CanUnlock returns true if the account can transition to ACTIVE.
func (s AccountStatus) CanUnlock() bool {
	return s == StatusLocked
}

// String returns the string representation.
func (s AccountStatus) String() string {
	return string(s)
}

// ParseAccountStatus converts a string to AccountStatus.
func ParseAccountStatus(s string) (AccountStatus, error) {
	switch s {
	case "ACTIVE":
		return StatusActive, nil
	case "LOCKED":
		return StatusLocked, nil
	default:
		return "", nil // or error
	}
}
