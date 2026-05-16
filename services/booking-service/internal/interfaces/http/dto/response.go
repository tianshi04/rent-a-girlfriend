package dto

import (
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
)

// --- Response DTOs ---

// BookingResponse is the JSON response for a single booking.
type BookingResponse struct {
	ID               string    `json:"id"`
	ClientID         string    `json:"clientId"`
	CompanionID      string    `json:"companionId"`
	ScenarioPrice    int       `json:"scenarioPrice"`
	ScenarioDuration int       `json:"scenarioDurationMinutes"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	Status           string    `json:"status"`
	CancelledByRole  string    `json:"cancelledByRole,omitempty"`
	IsLateCancel     bool      `json:"isLateCancel,omitempty"`
	Version          int       `json:"version"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// ListBookingsResponse is the paginated response for listing bookings.
type ListBookingsResponse struct {
	Data     []BookingResponse `json:"data"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ToBookingResponse maps a domain Booking to a response DTO.
func ToBookingResponse(b *aggregate.Booking) BookingResponse {
	return BookingResponse{
		ID:               b.ID().String(),
		ClientID:         b.ClientID().String(),
		CompanionID:      b.CompanionID().String(),
		ScenarioPrice:    b.Scenario().Price().Amount(),
		ScenarioDuration: b.Scenario().DurationMinutes(),
		StartTime:        b.TimeRange().StartTime(),
		EndTime:          b.TimeRange().EndTime(),
		Status:           string(b.Status()),
		CancelledByRole:  string(b.CancelledByRole()),
		IsLateCancel:     b.IsLateCancel(),
		Version:          b.Version(),
		CreatedAt:        b.CreatedAt(),
		UpdatedAt:        b.UpdatedAt(),
	}
}
