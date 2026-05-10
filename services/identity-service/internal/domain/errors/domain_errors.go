package errors

import "errors"

// Domain errors for the Identity Context.
var (
	ErrAccountNotFound        = errors.New("account not found")
	ErrAccountLocked          = errors.New("account is locked") // [INV-ID01]
	ErrAccountAlreadyActive   = errors.New("account is already active")
	ErrInvalidEmail           = errors.New("invalid email format")
	ErrInvalidRole            = errors.New("invalid role")
	ErrEmailAlreadyExists     = errors.New("email already registered")
	ErrConcurrencyConflict    = errors.New("concurrency conflict: account was modified by another process")
	ErrInvalidOAuthToken      = errors.New("invalid google oauth token")
	ErrAlreadyCompanion       = errors.New("account is already a companion")
	ErrUpgradeRequestPending  = errors.New("a pending upgrade request already exists")
	ErrUpgradeRequestNotFound = errors.New("upgrade request not found")
	ErrInvalidUpgradeStatus   = errors.New("invalid upgrade request status transition")
	ErrNotClient              = errors.New("only clients can request companion upgrade")
	ErrInvalidRefreshToken    = errors.New("invalid or expired refresh token")
	ErrRefreshTokenReuse      = errors.New("refresh token reuse detected")
	ErrPKCEStateNotFound      = errors.New("pkce state not found or expired")
)
