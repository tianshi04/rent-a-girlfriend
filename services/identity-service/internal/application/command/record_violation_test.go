package command

import (
	"context"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/service"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

func TestRecordViolationHandler_Handle(t *testing.T) {
	repo := &mockRepo{accounts: make(map[string]*aggregate.UserAccount)}
	configRepo := &mockConfigRepo{configs: make(map[string]string)}
	policySvc := service.NewAccountLockPolicyService(configRepo)
	pub := &mockPublisher{}
	
	handler := NewRecordViolationHandler(repo, policySvc, pub)

	email, _ := vo.NewEmail("user@example.com")
	acc := aggregate.NewUserAccount(email, "g1", time.Now())
	repo.accounts[acc.ID().String()] = acc

	cmd := RecordViolationCommand{
		UserID:    acc.ID().String(),
		Reason:    "Late cancellation",
		BookingID: "bk-999",
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if acc.ViolationCount() != 1 {
		t.Errorf("expected violation count 1, got %d", acc.ViolationCount())
	}
}
