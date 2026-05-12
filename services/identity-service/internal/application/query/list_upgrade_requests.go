package query

import (
	"context"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
)

// ListUpgradeRequestsQuery contains filter parameters.
type ListUpgradeRequestsQuery struct {
	Status   *string
	Page     int
	PageSize int
}

// ListUpgradeRequestsHandler lists upgrade requests with pagination.
type ListUpgradeRequestsHandler struct {
	upgradeRepo repository.UpgradeRequestRepository
}

// NewListUpgradeRequestsHandler creates a new handler.
func NewListUpgradeRequestsHandler(upgradeRepo repository.UpgradeRequestRepository) *ListUpgradeRequestsHandler {
	return &ListUpgradeRequestsHandler{upgradeRepo: upgradeRepo}
}

// Handle returns paginated upgrade requests.
func (h *ListUpgradeRequestsHandler) Handle(ctx context.Context, q ListUpgradeRequestsQuery) ([]*aggregate.UpgradeRequest, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	return h.upgradeRepo.FindByFilters(ctx, repository.UpgradeRequestFilters{
		Status:   q.Status,
		Page:     q.Page,
		PageSize: q.PageSize,
	})
}
