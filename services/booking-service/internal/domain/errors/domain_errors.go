package errors

import "errors"

// Domain errors for the Booking Context.
var (
	ErrBookingNotFound         = errors.New("booking not found")
	ErrInvalidTimeRange        = errors.New("invalid time range")
	ErrInvalidStatus           = errors.New("invalid status transition")
	ErrPendingCapExceeded      = errors.New("pending booking cap exceeded for this companion (max 10)")
	ErrConcurrencyConflict     = errors.New("concurrency conflict: booking was modified by another process")
	ErrUnauthorized            = errors.New("unauthorized: actor does not have permission")
	ErrInsufficientFunds       = errors.New("insufficient funds to freeze")
	ErrScenarioNotFound        = errors.New("scenario not found")
	ErrBookingAlreadyExist     = errors.New("booking already exists")
	ErrClientBookingOverlap    = errors.New("client already has an active or pending booking in this time range")
	ErrCompanionBookingOverlap = errors.New("companion already has an accepted booking in this time range")
)
