package command

import (
	"context"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

func TestLockAccountHandler_Handle(t *testing.T) {
	repo := &mockRepo{accounts: make(map[string]*aggregate.UserAccount)}
	pub := &mockPublisher{}
	tokenSvc := &mockTokenService{}
	handler := NewLockAccountHandler(repo, tokenSvc, pub)

	email, _ := vo.NewEmail("test@example.com")
	acc := aggregate.NewUserAccount(email, "google-1", time.Now())
	repo.accounts[acc.ID().String()] = acc

	cmd := LockAccountCommand{
		UserID:   acc.ID().String(),
		Reason:   "Policy violation",
		AdminID:  "admin-123",
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if acc.Status() != vo.StatusLocked {
		t.Errorf("expected status LOCKED, got %s", acc.Status())
	}

	if len(pub.events) != 2 {
		t.Errorf("expected 2 events published (Registered + Locked), got %d", len(pub.events))
	}
}
