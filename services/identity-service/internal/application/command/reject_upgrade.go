package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// RejectUpgradeCommand contains the upgrade request ID to reject.
type RejectUpgradeCommand struct {
	RequestID    string
	AdminID      string
	RejectReason string
}

// RejectUpgradeHandler rejects a companion upgrade request.
type RejectUpgradeHandler struct {
	upgradeRepo repository.UpgradeRequestRepository
	publisher   port.EventPublisher
}

// NewRejectUpgradeHandler creates a new handler.
func NewRejectUpgradeHandler(
	upgradeRepo repository.UpgradeRequestRepository,
	publisher port.EventPublisher,
) *RejectUpgradeHandler {
	return &RejectUpgradeHandler{
		upgradeRepo: upgradeRepo,
		publisher:   publisher,
	}
}

// Handle rejects the upgrade request.
func (h *RejectUpgradeHandler) Handle(ctx context.Context, cmd RejectUpgradeCommand) error {
	reqID, err := uuid.Parse(cmd.RequestID)
	if err != nil {
		return domainerr.ErrUpgradeRequestNotFound
	}

	req, err := h.upgradeRepo.FindByID(ctx, reqID)
	if err != nil {
		return domainerr.ErrUpgradeRequestNotFound
	}

	if err := req.Reject(cmd.AdminID, cmd.RejectReason, time.Now()); err != nil {
		return err
	}

	if err := h.upgradeRepo.Update(ctx, req); err != nil {
		return err
	}

	for _, evt := range req.Events() {
		if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
			return pubErr
		}
	}

	return nil
}
