package vo

import (
	"fmt"

	"github.com/google/uuid"
)

// ClientID references a Client in the Identity Context.
type ClientID struct {
	value uuid.UUID
}

func NewClientID(s string) (ClientID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ClientID{}, fmt.Errorf("invalid client id: %w", err)
	}
	return ClientID{value: id}, nil
}

func (c ClientID) String() string { return c.value.String() }
func (c ClientID) IsZero() bool   { return c.value == uuid.Nil }
func (c ClientID) Equals(other ClientID) bool {
	return c.value == other.value
}

// CompanionID references a Companion in the Identity Context.
type CompanionID struct {
	value uuid.UUID
}

func NewCompanionID(s string) (CompanionID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return CompanionID{}, fmt.Errorf("invalid companion id: %w", err)
	}
	return CompanionID{value: id}, nil
}

func (c CompanionID) String() string { return c.value.String() }
func (c CompanionID) IsZero() bool   { return c.value == uuid.Nil }
func (c CompanionID) Equals(other CompanionID) bool {
	return c.value == other.value
}

// ActorRole represents the role of the user performing an action.
type ActorRole string

const (
	RoleClient    ActorRole = "CLIENT"
	RoleCompanion ActorRole = "COMPANION"
)

func NewActorRole(s string) (ActorRole, error) {
	switch ActorRole(s) {
	case RoleClient, RoleCompanion:
		return ActorRole(s), nil
	default:
		return "", fmt.Errorf("invalid actor role: %s", s)
	}
}
