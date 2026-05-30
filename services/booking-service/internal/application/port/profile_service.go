package port

import (
	"context"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// ProfileService is the port for synchronous calls to Profile & Catalogue Context.
type ProfileService interface {
	// GetScenarioSnapshot fetches an immutable snapshot of a Scenario at booking time.
	GetScenarioSnapshot(ctx context.Context, scenarioID string) (*vo.ScenarioSnapshot, error)
}
