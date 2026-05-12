package command

import (
	"context"
	"time"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// UnlockAccountCommand contains the unlock parameters.
type UnlockAccountCommand struct {
	UserID  string
	AdminID string
}

// UnlockAccountHandler unlocks a user account (admin action).
type UnlockAccountHandler struct {
	accountRepo repository.UserAccountRepository
	publisher   port.EventPublisher
}

// NewUnlockAccountHandler creates a new handler.
func NewUnlockAccountHandler(
	accountRepo repository.UserAccountRepository,
	publisher port.EventPublisher,
) *UnlockAccountHandler {
	return &UnlockAccountHandler{accountRepo: accountRepo, publisher: publisher}
}

// Handle unlocks the given user account.
func (h *UnlockAccountHandler) Handle(ctx context.Context, cmd UnlockAccountCommand) error {
	userID, err := vo.ParseUserID(cmd.UserID)
	if err != nil {
		return err
	}

	account, err := h.accountRepo.FindByID(ctx, userID)
	if err != nil {
		return domainerr.ErrAccountNotFound
	}

	if err := account.Unlock(cmd.AdminID, time.Now()); err != nil {
		return err
	}

	if err := h.accountRepo.Update(ctx, account); err != nil {
		return err
	}

	for _, evt := range account.Events() {
		if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
			return pubErr
		}
	}

	return nil
}
