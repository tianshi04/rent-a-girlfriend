package command

import (
	"context"
	"time"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// RequestCompanionUpgradeCommand contains the upgrade request data.
type RequestCompanionUpgradeCommand struct {
	UserID string
	Reason string
}

// RequestCompanionUpgradeHandler handles CLIENT → COMPANION upgrade requests.
type RequestCompanionUpgradeHandler struct {
	accountRepo  repository.UserAccountRepository
	upgradeRepo  repository.UpgradeRequestRepository
	publisher    port.EventPublisher
}

// NewRequestCompanionUpgradeHandler creates a new handler.
func NewRequestCompanionUpgradeHandler(
	accountRepo repository.UserAccountRepository,
	upgradeRepo repository.UpgradeRequestRepository,
	publisher port.EventPublisher,
) *RequestCompanionUpgradeHandler {
	return &RequestCompanionUpgradeHandler{
		accountRepo: accountRepo,
		upgradeRepo: upgradeRepo,
		publisher:   publisher,
	}
}

// Handle creates a new companion upgrade request.
func (h *RequestCompanionUpgradeHandler) Handle(ctx context.Context, cmd RequestCompanionUpgradeCommand) error {
	userID, err := vo.ParseUserID(cmd.UserID)
	if err != nil {
		return err
	}

	// 1. Find user and validate role/status
	account, err := h.accountRepo.FindByID(ctx, userID)
	if err != nil {
		return domainerr.ErrAccountNotFound
	}

	if err := account.CheckLoginAllowed(); err != nil {
		return err
	}

	if !account.Role().CanUpgradeToCompanion() {
		return domainerr.ErrAlreadyCompanion
	}

	// 2. Check no pending request exists
	existing, err := h.upgradeRepo.FindPendingByUserID(ctx, userID)
	if err == nil && existing != nil {
		return domainerr.ErrUpgradeRequestPending
	}

	// 3. Create upgrade request
	req := aggregate.NewUpgradeRequest(userID, cmd.Reason, time.Now())
	if err := h.upgradeRepo.Save(ctx, req); err != nil {
		return err
	}

	// 4. Publish events
	for _, evt := range req.Events() {
		if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
			return pubErr
		}
	}

	return nil
}
