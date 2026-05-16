package vo

import "fmt"

// ScenarioSnapshot captures the price and duration at booking time.
// This ensures immutability even if the Companion changes pricing later.
type ScenarioSnapshot struct {
	price    Money
	duration int // in minutes
}

// NewScenarioSnapshot creates a validated ScenarioSnapshot.
func NewScenarioSnapshot(price Money, durationMinutes int) (ScenarioSnapshot, error) {
	if price.IsZero() {
		return ScenarioSnapshot{}, fmt.Errorf("scenario price must be greater than zero")
	}
	if durationMinutes <= 0 {
		return ScenarioSnapshot{}, fmt.Errorf("scenario duration must be positive")
	}
	return ScenarioSnapshot{price: price, duration: durationMinutes}, nil
}

// Price returns the snapshot price.
func (s ScenarioSnapshot) Price() Money { return s.price }

// DurationMinutes returns the duration in minutes.
func (s ScenarioSnapshot) DurationMinutes() int { return s.duration }
