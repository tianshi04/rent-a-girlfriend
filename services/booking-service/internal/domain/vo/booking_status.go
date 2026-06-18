package vo

// BookingStatus represents the current state of a Booking in the state machine.
type BookingStatus string

const (
	StatusPendingReserving BookingStatus = "PENDING_RESERVING"
	StatusPending          BookingStatus = "PENDING"
	StatusAccepted         BookingStatus = "ACCEPTED"
	StatusRejected         BookingStatus = "REJECTED"
	StatusCompleted        BookingStatus = "COMPLETED"
	StatusCancelled        BookingStatus = "CANCELLED"
	StatusDisputed         BookingStatus = "DISPUTED"
	StatusResolved         BookingStatus = "RESOLVED"
)

// CanAccept checks if the booking can transition to ACCEPTED. [INV-B03]
func (s BookingStatus) CanAccept() bool {
	return s == StatusPending
}

// CanReject checks if the booking can transition to REJECTED. [INV-B03]
func (s BookingStatus) CanReject() bool {
	return s == StatusPending
}

// CanCancel checks if the booking can transition to CANCELLED. [INV-B05]
func (s BookingStatus) CanCancel() bool {
	return s == StatusPending || s == StatusAccepted
}

// CanComplete checks if the booking can transition to COMPLETED.
func (s BookingStatus) CanComplete() bool {
	return s == StatusAccepted
}

// CanDispute checks if the booking can transition to DISPUTED.
func (s BookingStatus) CanDispute() bool {
	return s == StatusAccepted
}

// IsTerminal checks if the status is a terminal state.
func (s BookingStatus) IsTerminal() bool {
	return s == StatusRejected || s == StatusCompleted || s == StatusCancelled || s == StatusResolved
}
