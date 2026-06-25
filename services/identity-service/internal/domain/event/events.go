package event

import (
	"time"

	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
	"google.golang.org/protobuf/proto"
)

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	ToProto() proto.Message
}

// UserRegistered is raised when a new user account is created.
type UserRegistered struct {
	*identityv1.UserRegisteredPayload
}

func (e UserRegistered) EventType() string { return "identity.user-registered.v1" }
func (e UserRegistered) OccurredAt() time.Time {
	return e.UserRegisteredPayload.GetTimestamp().AsTime()
}
func (e UserRegistered) ToProto() proto.Message { return e.UserRegisteredPayload }

// ViolationRecorded is raised when a violation is recorded against an account.
type ViolationRecorded struct {
	*identityv1.ViolationRecordedPayload
	Occurred time.Time
}

func (e ViolationRecorded) EventType() string { return "identity.violation-recorded.v1" }
func (e ViolationRecorded) OccurredAt() time.Time {
	return e.Occurred
}
func (e ViolationRecorded) ToProto() proto.Message { return e.ViolationRecordedPayload }

// AccountLocked is raised when a user account is locked.
type AccountLocked struct {
	*identityv1.AccountLockedPayload
	Occurred time.Time
}

func (e AccountLocked) EventType() string      { return "identity.account-locked.v1" }
func (e AccountLocked) OccurredAt() time.Time  { return e.Occurred }
func (e AccountLocked) ToProto() proto.Message { return e.AccountLockedPayload }

// AccountUnlocked is raised when a user account is unlocked.
type AccountUnlocked struct {
	UserID     string    `json:"userId"`
	UnlockedBy string    `json:"unlockedBy"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e AccountUnlocked) EventType() string      { return "identity.account-unlocked.v1" }
func (e AccountUnlocked) OccurredAt() time.Time  { return e.Timestamp }
func (e AccountUnlocked) ToProto() proto.Message { return nil }

// CompanionUpgradeRequested is raised when a client requests to become a companion.
type CompanionUpgradeRequested struct {
	*identityv1.UpgradeRequestedPayload
	Occurred time.Time
}

func (e CompanionUpgradeRequested) EventType() string {
	return "identity.companion-upgrade-requested.v1"
}
func (e CompanionUpgradeRequested) OccurredAt() time.Time  { return e.Occurred }
func (e CompanionUpgradeRequested) ToProto() proto.Message { return e.UpgradeRequestedPayload }

// CompanionUpgradeApproved is raised when an admin approves a companion upgrade.
type CompanionUpgradeApproved struct {
	*identityv1.UpgradeApprovedPayload
	Occurred time.Time
}

func (e CompanionUpgradeApproved) EventType() string {
	return "identity.companion-upgrade-approved.v1"
}
func (e CompanionUpgradeApproved) OccurredAt() time.Time  { return e.Occurred }
func (e CompanionUpgradeApproved) ToProto() proto.Message { return e.UpgradeApprovedPayload }

// CompanionUpgradeRejected is raised when an admin rejects a companion upgrade.
type CompanionUpgradeRejected struct {
	*identityv1.UpgradeRejectedPayload
	Occurred time.Time
}

func (e CompanionUpgradeRejected) EventType() string {
	return "identity.companion-upgrade-rejected.v1"
}
func (e CompanionUpgradeRejected) OccurredAt() time.Time  { return e.Occurred }
func (e CompanionUpgradeRejected) ToProto() proto.Message { return e.UpgradeRejectedPayload }

// RoleUpgraded is raised when a user's role is upgraded from CLIENT to COMPANION.
type RoleUpgraded struct {
	UserID    string    `json:"userId"`
	OldRole   string    `json:"oldRole"`
	NewRole   string    `json:"newRole"`
	Timestamp time.Time `json:"timestamp"`
}

func (e RoleUpgraded) EventType() string      { return "identity.role-upgraded.v1" }
func (e RoleUpgraded) OccurredAt() time.Time  { return e.Timestamp }
func (e RoleUpgraded) ToProto() proto.Message { return nil }
