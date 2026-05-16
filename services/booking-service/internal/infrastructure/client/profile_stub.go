package client

import (
	"context"
	"log"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// ProfileServiceStub is a stub implementation of port.ProfileService.
// Returns mock data for Phase 1. Will be replaced by real gRPC/HTTP client.
type ProfileServiceStub struct{}

func NewProfileServiceStub() *ProfileServiceStub {
	return &ProfileServiceStub{}
}

func (s *ProfileServiceStub) GetScenarioSnapshot(ctx context.Context, scenarioID string) (*vo.ScenarioSnapshot, error) {
	log.Printf("[STUB] ProfileService.GetScenarioSnapshot called with scenarioID=%s", scenarioID)

	// Mock: return a default scenario with 500 Kano-Coin, 120 minutes
	price := vo.MustMoney(500)
	snapshot, err := vo.NewScenarioSnapshot(price, 120)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}
