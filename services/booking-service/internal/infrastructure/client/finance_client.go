package client

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	financev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/financev1"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// FinanceClient implements port.FinanceService using real gRPC client.
type FinanceClient struct {
	grpcClient financev1.FinanceServiceClient
}

func NewFinanceClient(cc grpc.ClientConnInterface) *FinanceClient {
	return &FinanceClient{
		grpcClient: financev1.NewFinanceServiceClient(cc),
	}
}

func (c *FinanceClient) FreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	log.Printf("[GRPC] FinanceService.FreezeCoin: clientID=%s, amount=%d", clientID.String(), amount.Amount())

	req := &financev1.FreezeCoinRequest{
		UserId: clientID.String(),
		Amount: amount.Amount(),
		Type:   bookingv1.TransactionType_TRANSACTION_TYPE_BOOKING_RESERVATION,
	}

	resp, err := c.grpcClient.FreezeCoin(propagateMetadata(ctx), req)
	if err != nil {
		return fmt.Errorf("failed to call FreezeCoin gRPC: %w", err)
	}

	if resp.Status != "SUCCESS" {
		return fmt.Errorf("failed to freeze coin: %s", resp.Message)
	}

	return nil
}

func (c *FinanceClient) UnfreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	log.Printf("[GRPC] FinanceService.UnfreezeCoin: clientID=%s, amount=%d", clientID.String(), amount.Amount())

	req := &financev1.FreezeCoinRequest{
		UserId: clientID.String(),
		Amount: -amount.Amount(), // negative amount to represent unfreezing
		Type:   bookingv1.TransactionType_TRANSACTION_TYPE_REFUND,
	}

	resp, err := c.grpcClient.FreezeCoin(propagateMetadata(ctx), req)
	if err != nil {
		return fmt.Errorf("failed to call FreezeCoin (unfreeze) gRPC: %w", err)
	}

	if resp.Status != "SUCCESS" {
		return fmt.Errorf("failed to unfreeze coin: %s", resp.Message)
	}

	return nil
}
