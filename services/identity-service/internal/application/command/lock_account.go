package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// LockAccountCommand contains the lock parameters.
type LockAccountCommand struct {
	UserID  string
	Reason  string
	AdminID string
}

// LockAccountHandler locks a user account (admin action).
type LockAccountHandler struct {
	accountRepo  repository.UserAccountRepository
	tokenService port.TokenService
	publisher    port.EventPublisher
}

// NewLockAccountHandler creates a new handler.
func NewLockAccountHandler(
	accountRepo repository.UserAccountRepository,
	tokenService port.TokenService,
	publisher port.EventPublisher,
) *LockAccountHandler {
	return &LockAccountHandler{
		accountRepo:  accountRepo,
		tokenService: tokenService,
		publisher:    publisher,
	}
}

// Handle locks the given user account.
func (h *LockAccountHandler) Handle(ctx context.Context, cmd LockAccountCommand) error {
	// 1. Validate Admin performing the action
	adminID, _ := vo.ParseUserID(cmd.AdminID)
	admin, err := h.accountRepo.FindByID(ctx, adminID)
	if err == nil { // If admin found in DB, ensure they are ACTIVE
		if err := admin.CheckLoginAllowed(); err != nil { // Reusing existing domain check
			return err
		}
	}

	userID, err := vo.ParseUserID(cmd.UserID)
	if err != nil {
		return err
	}

	// 2. Find and lock the target account
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

	// 3. REVOKE ALL TOKENS immediately to force logout
	_ = h.tokenService.RevokeAllUserTokens(userID)

	// 4. Publish events
	for _, evt := range account.Events() {
		if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
			return pubErr
		}
	}

	return nil
}
