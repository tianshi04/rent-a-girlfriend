package handler

import (
	"errors"
	"net/http"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
)

// mapDomainErrorToHTTP maps domain errors to appropriate HTTP status codes.
func mapDomainErrorToHTTP(err error) int {
	switch {
	case errors.Is(err, domainerr.ErrAccountNotFound),
		errors.Is(err, domainerr.ErrUpgradeRequestNotFound):
		return http.StatusNotFound

	case errors.Is(err, domainerr.ErrAccountLocked):
		return http.StatusForbidden

	case errors.Is(err, domainerr.ErrInvalidEmail),
		errors.Is(err, domainerr.ErrInvalidRole),
		errors.Is(err, domainerr.ErrInvalidUpgradeStatus),
		errors.Is(err, domainerr.ErrPKCEStateNotFound):
		return http.StatusBadRequest

	case errors.Is(err, domainerr.ErrEmailAlreadyExists),
		errors.Is(err, domainerr.ErrAlreadyCompanion),
		errors.Is(err, domainerr.ErrUpgradeRequestPending),
		errors.Is(err, domainerr.ErrNotClient),
		errors.Is(err, domainerr.ErrAccountAlreadyActive):
		return http.StatusConflict

	case errors.Is(err, domainerr.ErrConcurrencyConflict):
		return http.StatusConflict

	case errors.Is(err, domainerr.ErrInvalidOAuthToken),
		errors.Is(err, domainerr.ErrInvalidRefreshToken),
		errors.Is(err, domainerr.ErrRefreshTokenReuse):
		return http.StatusUnauthorized

	default:
		return http.StatusInternalServerError
	}
}
