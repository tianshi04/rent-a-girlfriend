package client

import (
	"context"
	"log"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// FinanceServiceStub is a stub implementation of port.FinanceService.
// Always returns success for Phase 1. Will be replaced by real gRPC client.
type FinanceServiceStub struct{}

func NewFinanceServiceStub() *FinanceServiceStub {
	return &FinanceServiceStub{}
}

func (s *FinanceServiceStub) FreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	log.Printf("[STUB] FinanceService.FreezeCoin called: clientID=%s, amount=%d", clientID.String(), amount.Amount())
	return nil
}

func (s *FinanceServiceStub) UnfreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	log.Printf("[STUB] FinanceService.UnfreezeCoin called: clientID=%s, amount=%d", clientID.String(), amount.Amount())
	return nil
}
