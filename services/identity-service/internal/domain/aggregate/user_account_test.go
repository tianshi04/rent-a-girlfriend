package aggregate

import (
	"testing"
	"time"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

func newTestAccount(t *testing.T) *UserAccount {
	t.Helper()
	email, err := vo.NewEmail("test@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}
	return NewUserAccount(email, "google-123", time.Now())
}

func TestNewUserAccount_DefaultsToClient(t *testing.T) {
	acc := newTestAccount(t)
	if acc.Role() != vo.RoleClient {
		t.Errorf("expected CLIENT, got %s", acc.Role())
	}
	if acc.Status() != vo.StatusActive {
		t.Errorf("expected ACTIVE, got %s", acc.Status())
	}
	if acc.ViolationCount() != 0 {
		t.Errorf("expected 0 violations, got %d", acc.ViolationCount())
	}

	events := acc.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType() != "com.rentagf.identity.UserRegistered.v1" {
		t.Errorf("unexpected event type: %s", events[0].EventType())
	}
}

func TestUserAccount_CheckLoginAllowed_Active(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events() // clear

	if err := acc.CheckLoginAllowed(); err != nil {
		t.Errorf("ACTIVE account should allow login: %v", err)
	}
}

func TestUserAccount_CheckLoginAllowed_Locked_INV_ID01(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()

	_ = acc.Lock("test reason", "admin-1", time.Now())
	_ = acc.Events()

	err := acc.CheckLoginAllowed()
	if err != domainerr.ErrAccountLocked {
		t.Errorf("[INV-ID01] expected ErrAccountLocked, got %v", err)
	}
}

func TestUserAccount_RecordViolation(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()

	acc.RecordViolation("no-show", "bk_001", time.Now())
	if acc.ViolationCount() != 1 {
		t.Errorf("expected 1 violation, got %d", acc.ViolationCount())
	}

	acc.RecordViolation("late-cancel", "bk_002", time.Now())
	if acc.ViolationCount() != 2 {
		t.Errorf("expected 2 violations, got %d", acc.ViolationCount())
	}

	events := acc.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestUserAccount_Lock_Success(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()

	err := acc.Lock("policy violation", "admin-1", time.Now())
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	if acc.Status() != vo.StatusLocked {
		t.Errorf("expected LOCKED, got %s", acc.Status())
	}

	events := acc.Events()
	if len(events) != 1 || events[0].EventType() != "com.rentagf.identity.AccountLocked.v1" {
		t.Error("expected AccountLocked event")
	}
}

func TestUserAccount_Lock_AlreadyLocked(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()
	_ = acc.Lock("reason", "admin-1", time.Now())
	_ = acc.Events()

	err := acc.Lock("reason2", "admin-1", time.Now())
	if err != domainerr.ErrAccountLocked {
		t.Errorf("expected ErrAccountLocked, got %v", err)
	}
}

func TestUserAccount_Unlock_Success(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()
	_ = acc.Lock("reason", "admin-1", time.Now())
	_ = acc.Events()

	err := acc.Unlock("admin-2", time.Now())
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	if acc.Status() != vo.StatusActive {
		t.Errorf("expected ACTIVE, got %s", acc.Status())
	}
}

func TestUserAccount_Unlock_AlreadyActive(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()

	err := acc.Unlock("admin-1", time.Now())
	if err != domainerr.ErrAccountAlreadyActive {
		t.Errorf("expected ErrAccountAlreadyActive, got %v", err)
	}
}

func TestUserAccount_UpgradeToCompanion_Success(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()

	err := acc.UpgradeToCompanion(time.Now())
	if err != nil {
		t.Fatalf("UpgradeToCompanion failed: %v", err)
	}
	if acc.Role() != vo.RoleCompanion {
		t.Errorf("expected COMPANION, got %s", acc.Role())
	}

	events := acc.Events()
	if len(events) != 1 || events[0].EventType() != "com.rentagf.identity.RoleUpgraded.v1" {
		t.Error("expected RoleUpgraded event")
	}
}

func TestUserAccount_UpgradeToCompanion_AlreadyCompanion(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Events()
	_ = acc.UpgradeToCompanion(time.Now())
	_ = acc.Events()

	err := acc.UpgradeToCompanion(time.Now())
	if err != domainerr.ErrAlreadyCompanion {
		t.Errorf("expected ErrAlreadyCompanion, got %v", err)
	}
}

func TestUserAccount_Unlock_NotLocked(t *testing.T) {
	acc := newTestAccount(t)
	err := acc.Unlock("admin-1", time.Now())
	if err != domainerr.ErrAccountAlreadyActive {
		t.Errorf("expected ErrAccountAlreadyActive when unlocking ACTIVE account, got %v", err)
	}
}

func TestUserAccount_Lock_Twice(t *testing.T) {
	acc := newTestAccount(t)
	_ = acc.Lock("first", "admin", time.Now())
	err := acc.Lock("second", "admin", time.Now())
	if err != domainerr.ErrAccountLocked {
		t.Errorf("expected ErrAccountLocked when locking already LOCKED account, got %v", err)
	}
}

// --- UpgradeRequest Tests ---

func TestUpgradeRequest_Approve(t *testing.T) {
	uid := vo.NewUserID()
	req := NewUpgradeRequest(uid, "I want to be a companion", time.Now())
	_ = req.Events()

	err := req.Approve("admin-1", time.Now())
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	if req.Status() != vo.UpgradeStatusApproved {
		t.Errorf("expected APPROVED, got %s", req.Status())
	}
}

func TestUpgradeRequest_Reject(t *testing.T) {
	uid := vo.NewUserID()
	req := NewUpgradeRequest(uid, "reason", time.Now())
	_ = req.Events()

	err := req.Reject("admin-1", "not qualified", time.Now())
	if err != nil {
		t.Fatalf("Reject failed: %v", err)
	}
	if req.Status() != vo.UpgradeStatusRejected {
		t.Errorf("expected REJECTED, got %s", req.Status())
	}
	if req.RejectReason() != "not qualified" {
		t.Errorf("unexpected reject reason: %s", req.RejectReason())
	}
}

func TestUpgradeRequest_ApproveAlreadyApproved(t *testing.T) {
	uid := vo.NewUserID()
	req := NewUpgradeRequest(uid, "reason", time.Now())
	_ = req.Events()
	_ = req.Approve("admin-1", time.Now())
	_ = req.Events()

	err := req.Approve("admin-2", time.Now())
	if err != domainerr.ErrInvalidUpgradeStatus {
		t.Errorf("expected ErrInvalidUpgradeStatus, got %v", err)
	}
}
