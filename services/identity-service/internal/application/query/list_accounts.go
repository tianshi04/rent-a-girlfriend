package query

import (
	"context"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
)

// ListAccountsQuery contains pagination parameters.
type ListAccountsQuery struct {
	Page     int
	PageSize int
}

// ListAccountsHandler retrieves a paginated list of user accounts.
type ListAccountsHandler struct {
	accountRepo repository.UserAccountRepository
}

// NewListAccountsHandler creates a new handler.
func NewListAccountsHandler(accountRepo repository.UserAccountRepository) *ListAccountsHandler {
	return &ListAccountsHandler{accountRepo: accountRepo}
}

// Handle retrieves the paginated user accounts.
func (h *ListAccountsHandler) Handle(ctx context.Context, q ListAccountsQuery) ([]*aggregate.UserAccount, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	return h.accountRepo.FindAll(ctx, q.Page, q.PageSize)
}
