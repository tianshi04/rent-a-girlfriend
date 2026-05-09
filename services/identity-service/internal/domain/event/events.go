package event

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// UserRegistered is raised when a new user account is created.
type UserRegistered struct {
	UserID    string    `json:"userId"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	GoogleID  string    `json:"googleId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e UserRegistered) EventType() string    { return "com.rentagf.identity.UserRegistered.v1" }
func (e UserRegistered) OccurredAt() time.Time { return e.Timestamp }

// ViolationRecorded is raised when a violation is recorded against an account.
type ViolationRecorded struct {
	UserID       string    `json:"userId"`
	CurrentCount int       `json:"currentCount"`
	Reason       string    `json:"reason"`
	BookingID    string    `json:"bookingId"`
	Timestamp    time.Time `json:"timestamp"`
}

func (e ViolationRecorded) EventType() string    { return "com.rentagf.identity.ViolationRecorded.v1" }
func (e ViolationRecorded) OccurredAt() time.Time { return e.Timestamp }

// AccountLocked is raised when a user account is locked.
type AccountLocked struct {
	UserID    string    `json:"userId"`
	Reason    string    `json:"reason"`
	LockedBy  string    `json:"lockedBy"`
	Timestamp time.Time `json:"timestamp"`
}

func (e AccountLocked) EventType() string    { return "com.rentagf.identity.AccountLocked.v1" }
func (e AccountLocked) OccurredAt() time.Time { return e.Timestamp }

// AccountUnlocked is raised when a user account is unlocked.
type AccountUnlocked struct {
	UserID     string    `json:"userId"`
	UnlockedBy string    `json:"unlockedBy"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e AccountUnlocked) EventType() string    { return "com.rentagf.identity.AccountUnlocked.v1" }
func (e AccountUnlocked) OccurredAt() time.Time { return e.Timestamp }

// CompanionUpgradeRequested is raised when a client requests to become a companion.
type CompanionUpgradeRequested struct {
	RequestID string    `json:"requestId"`
	UserID    string    `json:"userId"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (e CompanionUpgradeRequested) EventType() string {
	return "com.rentagf.identity.CompanionUpgradeRequested.v1"
}
func (e CompanionUpgradeRequested) OccurredAt() time.Time { return e.Timestamp }

// CompanionUpgradeApproved is raised when an admin approves a companion upgrade.
type CompanionUpgradeApproved struct {
	RequestID  string    `json:"requestId"`
	UserID     string    `json:"userId"`
	ApprovedBy string    `json:"approvedBy"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e CompanionUpgradeApproved) EventType() string {
	return "com.rentagf.identity.CompanionUpgradeApproved.v1"
}
func (e CompanionUpgradeApproved) OccurredAt() time.Time { return e.Timestamp }

// CompanionUpgradeRejected is raised when an admin rejects a companion upgrade.
type CompanionUpgradeRejected struct {
	RequestID    string    `json:"requestId"`
	UserID       string    `json:"userId"`
	RejectedBy   string    `json:"rejectedBy"`
	RejectReason string    `json:"rejectReason"`
	Timestamp    time.Time `json:"timestamp"`
}

func (e CompanionUpgradeRejected) EventType() string {
	return "com.rentagf.identity.CompanionUpgradeRejected.v1"
}
func (e CompanionUpgradeRejected) OccurredAt() time.Time { return e.Timestamp }

// RoleUpgraded is raised when a user's role is upgraded from CLIENT to COMPANION.
type RoleUpgraded struct {
	UserID    string    `json:"userId"`
	OldRole   string    `json:"oldRole"`
	NewRole   string    `json:"newRole"`
	Timestamp time.Time `json:"timestamp"`
}

func (e RoleUpgraded) EventType() string    { return "com.rentagf.identity.RoleUpgraded.v1" }
func (e RoleUpgraded) OccurredAt() time.Time { return e.Timestamp }
