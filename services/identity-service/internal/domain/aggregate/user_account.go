package aggregate

import (
	"time"

	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserAccount is the Aggregate Root for the Identity Context.
// All state changes must go through this aggregate to enforce invariants.
type UserAccount struct {
	id             vo.UserID
	email          vo.Email
	googleID       string
	role           vo.Role
	status         vo.AccountStatus
	violationCount int
	version        int
	createdAt      time.Time
	updatedAt      time.Time

	// uncommitted domain events collected during the lifecycle
	events []event.DomainEvent
}

// NewUserAccount creates a new UserAccount with default role CLIENT and status ACTIVE.
func NewUserAccount(email vo.Email, googleID string, name string, now time.Time) *UserAccount {
	a := &UserAccount{
		id:             vo.NewUserID(),
		email:          email,
		googleID:       googleID,
		role:           vo.RoleClient,
		status:         vo.StatusActive,
		violationCount: 0,
		version:        1,
		createdAt:      now,
		updatedAt:      now,
	}

	a.addEvent(event.UserRegistered{
		UserRegisteredPayload: &identityv1.UserRegisteredPayload{
			UserId:    a.id.String(),
			Email:     a.email.String(),
			Name:      name,
			Role:      string(a.role),
			GoogleId:  googleID,
			Timestamp: timestamppb.New(now),
		},
	})

	return a
}

// Reconstitute rebuilds a UserAccount from persistence (no validation, no events).
func Reconstitute(
	id vo.UserID,
	email vo.Email,
	googleID string,
	role vo.Role,
	status vo.AccountStatus,
	violationCount int,
	version int,
	createdAt, updatedAt time.Time,
) *UserAccount {
	return &UserAccount{
		id:             id,
		email:          email,
		googleID:       googleID,
		role:           role,
		status:         status,
		violationCount: violationCount,
		version:        version,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

// RecordViolation increments the violation counter.
func (a *UserAccount) RecordViolation(reason, bookingID string, now time.Time) {
	a.violationCount++
	a.updatedAt = now

	a.addEvent(event.ViolationRecorded{
		ViolationRecordedPayload: &identityv1.ViolationRecordedPayload{
			UserId:       a.id.String(),
			CurrentCount: int32(a.violationCount),
		},
		Occurred: now,
	})
}

// Lock transitions the account to LOCKED. [INV-ID01]
func (a *UserAccount) Lock(reason, lockedBy string, now time.Time) error {
	if !a.status.CanLock() {
		return domainerr.ErrAccountLocked
	}

	a.status = vo.StatusLocked
	a.updatedAt = now

	a.addEvent(event.AccountLocked{
		AccountLockedPayload: &identityv1.AccountLockedPayload{
			UserId: a.id.String(),
			Reason: reason,
		},
		Occurred: now,
	})
	return nil
}

// Unlock transitions the account from LOCKED to ACTIVE.
func (a *UserAccount) Unlock(unlockedBy string, now time.Time) error {
	if !a.status.CanUnlock() {
		return domainerr.ErrAccountAlreadyActive
	}

	a.status = vo.StatusActive
	a.updatedAt = now

	a.addEvent(event.AccountUnlocked{
		UserID:     a.id.String(),
		UnlockedBy: unlockedBy,
		Timestamp:  now,
	})
	return nil
}

// UpgradeToCompanion changes the role from CLIENT to COMPANION.
// Should only be called after admin has approved the upgrade request.
func (a *UserAccount) UpgradeToCompanion(now time.Time) error {
	if !a.role.CanUpgradeToCompanion() {
		return domainerr.ErrAlreadyCompanion
	}

	oldRole := a.role
	a.role = vo.RoleCompanion
	a.updatedAt = now

	a.addEvent(event.RoleUpgraded{
		UserID:    a.id.String(),
		OldRole:   string(oldRole),
		NewRole:   string(a.role),
		Timestamp: now,
	})
	return nil
}

// PromoteToAdmin forcefully changes the role to ADMIN.
// This bypasses the companion upgrade flow and should only be used
// for system bootstrapping or testing.
func (a *UserAccount) PromoteToAdmin() {
	oldRole := a.role
	a.role = vo.RoleAdmin
	a.updatedAt = time.Now()

	a.addEvent(event.RoleUpgraded{
		UserID:    a.id.String(),
		OldRole:   string(oldRole),
		NewRole:   string(a.role),
		Timestamp: a.updatedAt,
	})
}

// CheckLoginAllowed validates [INV-ID01]: cannot login if LOCKED.
func (a *UserAccount) CheckLoginAllowed() error {
	if !a.status.CanLogin() {
		return domainerr.ErrAccountLocked
	}
	return nil
}

// --- Getters ---

func (a *UserAccount) ID() vo.UserID            { return a.id }
func (a *UserAccount) Email() vo.Email          { return a.email }
func (a *UserAccount) GoogleID() string         { return a.googleID }
func (a *UserAccount) Role() vo.Role            { return a.role }
func (a *UserAccount) Status() vo.AccountStatus { return a.status }
func (a *UserAccount) ViolationCount() int      { return a.violationCount }
func (a *UserAccount) Version() int             { return a.version }
func (a *UserAccount) CreatedAt() time.Time     { return a.createdAt }
func (a *UserAccount) UpdatedAt() time.Time     { return a.updatedAt }

// Events returns uncommitted domain events and clears the internal list.
func (a *UserAccount) Events() []event.DomainEvent {
	events := a.events
	a.events = nil
	return events
}

// --- Private helpers ---

func (a *UserAccount) addEvent(e event.DomainEvent) {
	a.events = append(a.events, e)
}
