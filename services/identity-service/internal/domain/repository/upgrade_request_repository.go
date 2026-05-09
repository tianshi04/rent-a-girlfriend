package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// UpgradeRequestRepository is the port interface for persisting UpgradeRequest entities.
type UpgradeRequestRepository interface {
	// Save persists a new UpgradeRequest.
	Save(ctx context.Context, req *aggregate.UpgradeRequest) error

	// Update persists changes to an existing UpgradeRequest.
	Update(ctx context.Context, req *aggregate.UpgradeRequest) error

	// FindByID retrieves an UpgradeRequest by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*aggregate.UpgradeRequest, error)

	// FindPendingByUserID retrieves the pending upgrade request for a user.
	FindPendingByUserID(ctx context.Context, userID vo.UserID) (*aggregate.UpgradeRequest, error)

	// FindByFilters retrieves upgrade requests with optional filtering and pagination.
	FindByFilters(ctx context.Context, filters UpgradeRequestFilters) ([]*aggregate.UpgradeRequest, int64, error)
}

// UpgradeRequestFilters contains optional filter criteria for listing upgrade requests.
type UpgradeRequestFilters struct {
	Status   *string
	Page     int
	PageSize int
}
