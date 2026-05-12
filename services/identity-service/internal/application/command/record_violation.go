package command

import (
	"context"
	"time"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/service"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// RecordViolationCommand contains the violation data from Dispute Context.
type RecordViolationCommand struct {
	UserID    string
	Reason    string
	BookingID string
}

// RecordViolationHandler records a violation and auto-locks if threshold is reached.
type RecordViolationHandler struct {
	accountRepo repository.UserAccountRepository
	lockPolicy  *service.AccountLockPolicyService
	publisher   port.EventPublisher
}

// NewRecordViolationHandler creates a new handler.
func NewRecordViolationHandler(
	accountRepo repository.UserAccountRepository,
	lockPolicy *service.AccountLockPolicyService,
	publisher port.EventPublisher,
) *RecordViolationHandler {
	return &RecordViolationHandler{
		accountRepo: accountRepo,
		lockPolicy:  lockPolicy,
		publisher:   publisher,
	}
}

// Handle records a violation on the user account and auto-locks if policy dictates.
func (h *RecordViolationHandler) Handle(ctx context.Context, cmd RecordViolationCommand) error {
	userID, err := vo.ParseUserID(cmd.UserID)
	if err != nil {
		return err
	}

	account, err := h.accountRepo.FindByID(ctx, userID)
	if err != nil {
		return domainerr.ErrAccountNotFound
	}

	now := time.Now()

	// 1. Record violation
	account.RecordViolation(cmd.Reason, cmd.BookingID, now)

	// 2. Check auto-lock policy
	shouldLock, err := h.lockPolicy.ShouldLock(ctx, account.ViolationCount())
	if err != nil {
		return err
	}

	if shouldLock && account.Status().CanLock() {
		if lockErr := account.Lock("auto-lock: violation threshold reached", "system", now); lockErr != nil {
			return lockErr
		}
	}

	// 3. Persist
	if err := h.accountRepo.Update(ctx, account); err != nil {
		return err
	}

	// 4. Publish events
	for _, evt := range account.Events() {
		if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
			return pubErr
		}
	}

	return nil
}
