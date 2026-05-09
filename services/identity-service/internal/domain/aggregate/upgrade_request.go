package aggregate

import (
	"time"

	"github.com/google/uuid"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// UpgradeRequest represents a client's request to become a companion.
// Managed as part of the Identity bounded context.
type UpgradeRequest struct {
	id           uuid.UUID
	userID       vo.UserID
	status       vo.UpgradeStatus
	reason       string
	rejectReason string
	reviewedBy   string
	reviewedAt   *time.Time
	createdAt    time.Time

	events []event.DomainEvent
}

// NewUpgradeRequest creates a new upgrade request in PENDING status.
func NewUpgradeRequest(userID vo.UserID, reason string, now time.Time) *UpgradeRequest {
	r := &UpgradeRequest{
		id:        uuid.New(),
		userID:    userID,
		status:    vo.UpgradeStatusPending,
		reason:    reason,
		createdAt: now,
	}

	r.addEvent(event.CompanionUpgradeRequested{
		RequestID: r.id.String(),
		UserID:    userID.String(),
		Reason:    reason,
		Timestamp: now,
	})

	return r
}

// ReconstituteUpgradeRequest rebuilds from persistence (no validation, no events).
func ReconstituteUpgradeRequest(
	id uuid.UUID,
	userID vo.UserID,
	status vo.UpgradeStatus,
	reason, rejectReason, reviewedBy string,
	reviewedAt *time.Time,
	createdAt time.Time,
) *UpgradeRequest {
	return &UpgradeRequest{
		id:           id,
		userID:       userID,
		status:       status,
		reason:       reason,
		rejectReason: rejectReason,
		reviewedBy:   reviewedBy,
		reviewedAt:   reviewedAt,
		createdAt:    createdAt,
	}
}

// Approve transitions the request to APPROVED.
func (r *UpgradeRequest) Approve(adminID string, now time.Time) error {
	if !r.status.CanApprove() {
		return domainerr.ErrInvalidUpgradeStatus
	}

	r.status = vo.UpgradeStatusApproved
	r.reviewedBy = adminID
	r.reviewedAt = &now

	r.addEvent(event.CompanionUpgradeApproved{
		RequestID:  r.id.String(),
		UserID:     r.userID.String(),
		ApprovedBy: adminID,
		Timestamp:  now,
	})
	return nil
}

// Reject transitions the request to REJECTED.
func (r *UpgradeRequest) Reject(adminID, rejectReason string, now time.Time) error {
	if !r.status.CanReject() {
		return domainerr.ErrInvalidUpgradeStatus
	}

	r.status = vo.UpgradeStatusRejected
	r.rejectReason = rejectReason
	r.reviewedBy = adminID
	r.reviewedAt = &now

	r.addEvent(event.CompanionUpgradeRejected{
		RequestID:    r.id.String(),
		UserID:       r.userID.String(),
		RejectedBy:   adminID,
		RejectReason: rejectReason,
		Timestamp:    now,
	})
	return nil
}

// --- Getters ---

func (r *UpgradeRequest) ID() uuid.UUID            { return r.id }
func (r *UpgradeRequest) UserID() vo.UserID         { return r.userID }
func (r *UpgradeRequest) Status() vo.UpgradeStatus  { return r.status }
func (r *UpgradeRequest) Reason() string            { return r.reason }
func (r *UpgradeRequest) RejectReason() string      { return r.rejectReason }
func (r *UpgradeRequest) ReviewedBy() string        { return r.reviewedBy }
func (r *UpgradeRequest) ReviewedAt() *time.Time    { return r.reviewedAt }
func (r *UpgradeRequest) CreatedAt() time.Time      { return r.createdAt }

// Events returns uncommitted domain events and clears the internal list.
func (r *UpgradeRequest) Events() []event.DomainEvent {
	events := r.events
	r.events = nil
	return events
}

func (r *UpgradeRequest) addEvent(e event.DomainEvent) {
	r.events = append(r.events, e)
}
