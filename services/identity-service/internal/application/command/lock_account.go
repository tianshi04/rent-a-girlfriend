package command

import (
	"context"
	"time"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// LockAccountCommand contains the lock parameters.
type LockAccountCommand struct {
	UserID string
	Reason string
	AdminID string
}

// LockAccountHandler locks a user account (admin action).
type LockAccountHandler struct {
	accountRepo repository.UserAccountRepository
	publisher   port.EventPublisher
}

// NewLockAccountHandler creates a new handler.
func NewLockAccountHandler(
	accountRepo repository.UserAccountRepository,
	publisher port.EventPublisher,
) *LockAccountHandler {
	return &LockAccountHandler{accountRepo: accountRepo, publisher: publisher}
}

// Handle locks the given user account.
func (h *LockAccountHandler) Handle(ctx context.Context, cmd LockAccountCommand) error {
	userID, err := vo.ParseUserID(cmd.UserID)
	if err != nil {
		return err
	}

	account, err := h.accountRepo.FindByID(ctx, userID)
	if err != nil {
		return domainerr.ErrAccountNotFound
	}

	if err := account.Lock(cmd.Reason, cmd.AdminID, time.Now()); err != nil {
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
