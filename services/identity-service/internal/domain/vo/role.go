package vo

import (
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
)

// Role represents a user's role on the platform.
type Role string

const (
	RoleClient    Role = "CLIENT"
	RoleCompanion Role = "COMPANION"
	RoleAdmin     Role = "ADMIN"
)

// ParseRole converts a string to a valid Role.
func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleClient, RoleCompanion, RoleAdmin:
		return Role(s), nil
	default:
		return "", domainerr.ErrInvalidRole
	}
}

// IsValid checks if the role is one of the known values.
func (r Role) IsValid() bool {
	switch r {
	case RoleClient, RoleCompanion, RoleAdmin:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (r Role) String() string {
	return string(r)
}

// CanUpgradeToCompanion returns true if this role can request a companion upgrade.
func (r Role) CanUpgradeToCompanion() bool {
	return r == RoleClient
}
