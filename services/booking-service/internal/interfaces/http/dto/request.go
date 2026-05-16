package dto

import "time"

// --- Request DTOs ---

// RequestBookingRequest is the JSON body for creating a booking.
type RequestBookingRequest struct {
	ClientID    string    `json:"clientId" binding:"required"`
	CompanionID string    `json:"companionId" binding:"required"`
	ScenarioID  string    `json:"scenarioId" binding:"required"`
	StartTime   time.Time `json:"startTime" binding:"required"`
}

// AcceptBookingRequest is the JSON body for accepting a booking.
type AcceptBookingRequest struct {
	CompanionID string `json:"companionId" binding:"required"`
}

// RejectBookingRequest is the JSON body for rejecting a booking.
type RejectBookingRequest struct {
	CompanionID string `json:"companionId" binding:"required"`
}

// CancelBookingRequest is the JSON body for cancelling a booking.
type CancelBookingRequest struct {
	ActorID   string `json:"actorId" binding:"required"`
	ActorRole string `json:"actorRole" binding:"required"`
}

// ListBookingsRequest holds query params for listing bookings.
type ListBookingsRequest struct {
	ClientID    *string `form:"clientId"`
	CompanionID *string `form:"companionId"`
	Status      *string `form:"status"`
	Page        int     `form:"page,default=1"`
	PageSize    int     `form:"pageSize,default=20"`
}
