package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// ApproveUpgradeCommand contains the upgrade request ID to approve.
type ApproveUpgradeCommand struct {
	RequestID string
	AdminID   string
}

// ApproveUpgradeHandler approves a companion upgrade request and upgrades the user's role.
type ApproveUpgradeHandler struct {
	upgradeRepo repository.UpgradeRequestRepository
	accountRepo repository.UserAccountRepository
	publisher   port.EventPublisher
}

// NewApproveUpgradeHandler creates a new handler.
func NewApproveUpgradeHandler(
	upgradeRepo repository.UpgradeRequestRepository,
	accountRepo repository.UserAccountRepository,
	publisher port.EventPublisher,
) *ApproveUpgradeHandler {
	return &ApproveUpgradeHandler{
		upgradeRepo: upgradeRepo,
		accountRepo: accountRepo,
		publisher:   publisher,
	}
}

// Handle approves the upgrade request and changes user role to COMPANION.
func (h *ApproveUpgradeHandler) Handle(ctx context.Context, cmd ApproveUpgradeCommand) error {
	reqID, err := uuid.Parse(cmd.RequestID)
	if err != nil {
		return domainerr.ErrUpgradeRequestNotFound
	}

	now := time.Now()

	// 1. Find and approve the upgrade request
	req, err := h.upgradeRepo.FindByID(ctx, reqID)
	if err != nil {
		return domainerr.ErrUpgradeRequestNotFound
	}

	if err := req.Approve(cmd.AdminID, now); err != nil {
		return err
	}

	if err := h.upgradeRepo.Update(ctx, req); err != nil {
		return err
	}

	// 2. Upgrade user account role
	account, err := h.accountRepo.FindByID(ctx, req.UserID())
	if err != nil {
		return domainerr.ErrAccountNotFound
	}

	if err := account.UpgradeToCompanion(now); err != nil {
		return err
	}

	if err := h.accountRepo.Update(ctx, account); err != nil {
		return err
	}

	// 3. Publish all events
	for _, evt := range req.Events() {
		if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
			return pubErr
		}
	}
	for _, evt := range account.Events() {
		if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
			return pubErr
		}
	}

	return nil
}
