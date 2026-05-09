package query

import (
	"context"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// GetAccountHandler retrieves a user account by ID.
type GetAccountHandler struct {
	accountRepo repository.UserAccountRepository
}

// NewGetAccountHandler creates a new handler.
func NewGetAccountHandler(accountRepo repository.UserAccountRepository) *GetAccountHandler {
	return &GetAccountHandler{accountRepo: accountRepo}
}

// Handle retrieves the user account.
func (h *GetAccountHandler) Handle(ctx context.Context, userIDStr string) (*aggregate.UserAccount, error) {
	userID, err := vo.ParseUserID(userIDStr)
	if err != nil {
		return nil, domainerr.ErrAccountNotFound
	}
	return h.accountRepo.FindByID(ctx, userID)
}
