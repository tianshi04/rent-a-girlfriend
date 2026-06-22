package aggregate

import (
	"time"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Booking is the Aggregate Root for the Booking Context.
// All state changes must go through this aggregate to enforce invariants.
type Booking struct {
	id              vo.BookingID
	clientID        vo.ClientID
	companionID     vo.CompanionID
	scenario        vo.ScenarioSnapshot
	timeRange       vo.TimeRange
	status          vo.BookingStatus
	cancelledByRole vo.ActorRole
	isLateCancel    bool
	version         int64
	createdAt       time.Time
	updatedAt       time.Time

	// uncommitted domain events collected during the lifecycle
	events []event.DomainEvent
}

// NewBooking creates a new Booking in PENDING status.
// Validates INV-B01 and INV-B02.
func NewBooking(
	clientID vo.ClientID,
	companionID vo.CompanionID,
	scenario vo.ScenarioSnapshot,
	timeRange vo.TimeRange,
	now time.Time,
) (*Booking, error) {
	// [INV-B01] StartTime must be at least 2 hours from now
	if err := timeRange.ValidateAdvanceBooking(now); err != nil {
		return nil, err
	}

	// [INV-B02] EndTime must equal StartTime + Scenario.Duration
	if err := timeRange.ValidateDuration(scenario.DurationMinutes()); err != nil {
		return nil, err
	}

	b := &Booking{
		id:          vo.NewBookingID(),
		clientID:    clientID,
		companionID: companionID,
		scenario:    scenario,
		timeRange:   timeRange,
		status:      vo.StatusPendingReserving,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
	}

	b.addEvent(event.BookingRequested{
		BookingRequested: &bookingv1.BookingRequested{
			BookingId:   b.id.String(),
			ClientId:    b.clientID.String(),
			CompanionId: b.companionID.String(),
			Price:       b.scenario.Price().Amount(),
			StartTime:   timestamppb.New(b.timeRange.StartTime()),
			OccurredAt:  timestamppb.New(now),
		},
	})

	return b, nil
}

// Reconstitute rebuilds a Booking from persistence (no validation, no events).
func Reconstitute(
	id vo.BookingID,
	clientID vo.ClientID,
	companionID vo.CompanionID,
	scenario vo.ScenarioSnapshot,
	timeRange vo.TimeRange,
	status vo.BookingStatus,
	cancelledByRole vo.ActorRole,
	isLateCancel bool,
	version int64,
	createdAt, updatedAt time.Time,
) *Booking {
	return &Booking{
		id:              id,
		clientID:        clientID,
		companionID:     companionID,
		scenario:        scenario,
		timeRange:       timeRange,
		status:          status,
		cancelledByRole: cancelledByRole,
		isLateCancel:    isLateCancel,
		version:         version,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

// Accept transitions the booking to ACCEPTED. [INV-B03]
func (b *Booking) Accept(companionID vo.CompanionID, now time.Time) error {
	if !b.companionID.Equals(companionID) {
		return domainerr.ErrUnauthorized
	}
	if !b.status.CanAccept() {
		return domainerr.ErrInvalidStatus
	}

	b.status = vo.StatusAccepted
	b.updatedAt = now

	b.addEvent(event.BookingAccepted{
		BookingAccepted: &bookingv1.BookingAccepted{
			BookingId:   b.id.String(),
			CompanionId: b.companionID.String(),
			OccurredAt:  timestamppb.New(now),
		},
	})
	return nil
}

// Reject transitions the booking to REJECTED. [INV-B03]
func (b *Booking) Reject(companionID vo.CompanionID, reason string, now time.Time) error {
	if !b.companionID.Equals(companionID) {
		return domainerr.ErrUnauthorized
	}
	if !b.status.CanReject() {
		return domainerr.ErrInvalidStatus
	}

	b.status = vo.StatusRejected
	b.updatedAt = now

	b.addEvent(event.BookingRejected{
		BookingRejected: &bookingv1.BookingRejected{
			BookingId:   b.id.String(),
			CompanionId: b.companionID.String(),
			ClientId:    b.clientID.String(),
			Reason:      reason,
			OccurredAt:  timestamppb.New(now),
		},
	})
	return nil
}

// Cancel transitions the booking to CANCELLED. [INV-B05]
// Determines early/late cancellation based on 24h threshold (BR-05/06/07).
func (b *Booking) Cancel(actorRole vo.ActorRole, now time.Time) error {
	if !b.status.CanCancel() {
		return domainerr.ErrInvalidStatus
	}

	originalStatus := b.status
	b.status = vo.StatusCancelled
	b.cancelledByRole = actorRole

	if originalStatus == vo.StatusPending {
		b.isLateCancel = false
	} else {
		b.isLateCancel = b.timeRange.IsLateCancel(now)
	}

	b.updatedAt = now

	if b.isLateCancel {
		b.addEvent(event.BookingCancelledLate{
			BookingCancelledLate: &bookingv1.BookingCancelledLate{
				BookingId:  b.id.String(),
				ActorId:    b.resolveActorID(actorRole),
				ActorRole:  string(actorRole),
				OccurredAt: timestamppb.New(now),
			},
		})
	} else {
		b.addEvent(event.BookingCancelledEarly{
			BookingCancelledEarly: &bookingv1.BookingCancelledEarly{
				BookingId:  b.id.String(),
				ActorId:    b.resolveActorID(actorRole),
				ActorRole:  string(actorRole),
				OccurredAt: timestamppb.New(now),
			},
		})
	}
	return nil
}

// Complete transitions the booking to COMPLETED.
func (b *Booking) Complete(clientID vo.ClientID, now time.Time) error {
	if !b.clientID.Equals(clientID) {
		return domainerr.ErrUnauthorized
	}
	if !b.status.CanComplete() {
		return domainerr.ErrInvalidStatus
	}

	b.status = vo.StatusCompleted
	b.updatedAt = now

	b.addEvent(event.BookingCompleted{
		BookingCompleted: &bookingv1.BookingCompleted{
			BookingId:   b.id.String(),
			CompanionId: b.companionID.String(),
			ClientId:    b.clientID.String(),
			Price:       b.scenario.Price().Amount(),
			OccurredAt:  timestamppb.New(now),
		},
	})
	return nil
}

// SystemComplete transitions the booking to COMPLETED automatically (e.g. past end_time + buffer).
func (b *Booking) SystemComplete(now time.Time) error {
	if !b.status.CanComplete() {
		return domainerr.ErrInvalidStatus
	}

	b.status = vo.StatusCompleted
	b.updatedAt = now

	b.addEvent(event.BookingCompleted{
		BookingCompleted: &bookingv1.BookingCompleted{
			BookingId:   b.id.String(),
			CompanionId: b.companionID.String(),
			ClientId:    b.clientID.String(),
			Price:       b.scenario.Price().Amount(),
			OccurredAt:  timestamppb.New(now),
		},
	})
	return nil
}

// SystemTimeout transitions the booking to REJECTED due to timeout.
func (b *Booking) SystemTimeout(now time.Time) error {
	if !b.status.CanReject() {
		return domainerr.ErrInvalidStatus
	}

	b.status = vo.StatusRejected
	b.updatedAt = now

	b.addEvent(event.BookingTimedOut{
		BookingTimedOut: &bookingv1.BookingTimedOut{
			BookingId:  b.id.String(),
			ClientId:   b.clientID.String(),
			OccurredAt: timestamppb.New(now),
		},
	})
	return nil
}

// Dispute transitions the booking to DISPUTED state (locks it from auto-completion).
func (b *Booking) Dispute(now time.Time) error {
	if !b.status.CanDispute() {
		return domainerr.ErrInvalidStatus
	}

	b.status = vo.StatusDisputed
	b.updatedAt = now

	return nil
}

// Resolve transitions the booking from DISPUTED to RESOLVED status.
func (b *Booking) Resolve(now time.Time) error {
	if b.status != vo.StatusDisputed {
		return domainerr.ErrInvalidStatus
	}

	b.status = vo.StatusResolved
	b.updatedAt = now

	return nil
}

// FailTechnical transitions the booking to CANCELLED state due to tech failure (e.g. escrow failure)
func (b *Booking) FailTechnical(now time.Time) {
	b.status = vo.StatusCancelled
	b.cancelledByRole = vo.ActorRole("SYSTEM")
	b.isLateCancel = false
	b.updatedAt = now
}

// ConfirmReserved transitions the booking from PENDING_RESERVING to PENDING.
func (b *Booking) ConfirmReserved(now time.Time) error {
	if b.status != vo.StatusPendingReserving {
		return domainerr.ErrInvalidStatus
	}
	b.status = vo.StatusPending
	b.updatedAt = now

	b.addEvent(event.BookingReserved{
		BookingReserved: &bookingv1.BookingReserved{
			BookingId:   b.id.String(),
			CompanionId: b.companionID.String(),
			ClientId:    b.clientID.String(),
			OccurredAt:  timestamppb.New(now),
		},
	})
	return nil
}

// CancelReserving transitions the booking from PENDING_RESERVING to CANCELLED (due to freeze coin failure).
func (b *Booking) CancelReserving(reason string, now time.Time) error {
	if b.status != vo.StatusPendingReserving {
		return domainerr.ErrInvalidStatus
	}
	b.status = vo.StatusCancelled
	b.cancelledByRole = vo.ActorRole("SYSTEM")
	b.isLateCancel = false
	b.updatedAt = now
	return nil
}

// --- Getters ---

func (b *Booking) ID() vo.BookingID              { return b.id }
func (b *Booking) ClientID() vo.ClientID         { return b.clientID }
func (b *Booking) CompanionID() vo.CompanionID   { return b.companionID }
func (b *Booking) Scenario() vo.ScenarioSnapshot { return b.scenario }
func (b *Booking) TimeRange() vo.TimeRange       { return b.timeRange }
func (b *Booking) Status() vo.BookingStatus      { return b.status }
func (b *Booking) CancelledByRole() vo.ActorRole { return b.cancelledByRole }
func (b *Booking) IsLateCancel() bool            { return b.isLateCancel }
func (b *Booking) Version() int64                { return b.version }
func (b *Booking) CreatedAt() time.Time          { return b.createdAt }
func (b *Booking) UpdatedAt() time.Time          { return b.updatedAt }

// Events returns uncommitted domain events and clears the internal list.
func (b *Booking) Events() []event.DomainEvent {
	events := b.events
	b.events = nil
	return events
}

// --- Private helpers ---

func (b *Booking) addEvent(e event.DomainEvent) {
	b.events = append(b.events, e)
}

func (b *Booking) resolveActorID(role vo.ActorRole) string {
	if role == vo.RoleClient {
		return b.clientID.String()
	}
	if string(role) == "SYSTEM" {
		return "SYSTEM"
	}
	return b.companionID.String()
}
