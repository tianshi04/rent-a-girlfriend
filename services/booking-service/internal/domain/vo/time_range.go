package vo

import (
	"fmt"
	"time"
)

// TimeRange encapsulates StartTime and EndTime with validation.
// [INV-B01] StartTime must be at least 2 hours from now.
// [INV-B02] EndTime must equal StartTime + Scenario.Duration.
type TimeRange struct {
	startTime time.Time
	endTime   time.Time
}

// MinAdvanceBookingHours is the minimum hours ahead for booking [INV-B01].
const MinAdvanceBookingHours = 2

// NewTimeRange creates a validated TimeRange.
func NewTimeRange(startTime, endTime time.Time) (TimeRange, error) {
	if startTime.IsZero() || endTime.IsZero() {
		return TimeRange{}, fmt.Errorf("start time and end time must not be zero")
	}
	if !endTime.After(startTime) {
		return TimeRange{}, fmt.Errorf("end time must be after start time")
	}
	return TimeRange{startTime: startTime, endTime: endTime}, nil
}

// ValidateAdvanceBooking checks INV-B01: StartTime > now + 2h.
func (t TimeRange) ValidateAdvanceBooking(now time.Time) error {
	minStart := now.Add(time.Duration(MinAdvanceBookingHours) * time.Hour)
	if t.startTime.Before(minStart) {
		return fmt.Errorf("start time must be at least %d hours from now", MinAdvanceBookingHours)
	}
	return nil
}

// ValidateDuration checks INV-B02: duration matches scenario duration.
func (t TimeRange) ValidateDuration(scenarioDurationMinutes int) error {
	expected := time.Duration(scenarioDurationMinutes) * time.Minute
	actual := t.endTime.Sub(t.startTime)
	if actual != expected {
		return fmt.Errorf("time range duration (%v) does not match scenario duration (%v)", actual, expected)
	}
	return nil
}

// StartTime returns the start time.
func (t TimeRange) StartTime() time.Time { return t.startTime }

// EndTime returns the end time.
func (t TimeRange) EndTime() time.Time { return t.endTime }

// IsLateCancel checks if cancellation is within 24h of start time (BR-05/BR-06).
func (t TimeRange) IsLateCancel(now time.Time) bool {
	return now.Add(24 * time.Hour).After(t.startTime)
}
