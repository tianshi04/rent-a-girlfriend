package client

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	profilev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/profilev1"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// ProfileClient implements port.ProfileService using real gRPC client.
type ProfileClient struct {
	grpcClient profilev1.ProfileServiceClient
}

func NewProfileClient(cc grpc.ClientConnInterface) *ProfileClient {
	return &ProfileClient{
		grpcClient: profilev1.NewProfileServiceClient(cc),
	}
}

func (c *ProfileClient) GetScenarioSnapshot(ctx context.Context, scenarioID string) (*vo.ScenarioSnapshot, error) {
	log.Printf("[GRPC] ProfileService.GetScenarioSnapshot: scenarioID=%s", scenarioID)

	req := &profilev1.GetScenarioSnapshotRequest{
		ScenarioId: scenarioID,
	}

	resp, err := c.grpcClient.GetScenarioSnapshot(propagateMetadata(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetScenarioSnapshot gRPC: %w", err)
	}

	price, err := vo.NewMoney(resp.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price from profile service: %w", err)
	}

	snapshot, err := vo.NewScenarioSnapshot(price, int64(resp.DurationMinutes))
	if err != nil {
		return nil, fmt.Errorf("failed to build scenario snapshot: %w", err)
	}

	return &snapshot, nil
}

func propagateMetadata(ctx context.Context) context.Context {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		outgoingMd := metadata.New(map[string]string{})
		for _, key := range []string{"user-id", "user-role", "user-status", "user-email"} {
			if values := md.Get(key); len(values) > 0 {
				outgoingMd.Set(key, values[0])
			}
		}
		return metadata.NewOutgoingContext(ctx, outgoingMd)
	}
	return ctx
}
