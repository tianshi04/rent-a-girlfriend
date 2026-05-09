package vo

import (
	"strings"
	"testing"
)

func TestNewEmail_Valid(t *testing.T) {
	cases := []string{
		"user@example.com",
		"USER@Example.COM",
		"test.name+tag@domain.org",
	}
	for _, tc := range cases {
		email, err := NewEmail(tc)
		if err != nil {
			t.Errorf("NewEmail(%q) returned error: %v", tc, err)
		}
		// Should be normalized to lowercase
		if email.String() != strings.ToLower(strings.TrimSpace(tc)) {
			t.Errorf("expected normalized email, got %q", email.String())
		}
	}
}

func TestNewEmail_Invalid(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"not-an-email",
		"@domain.com",
		"user@",
		"user@.com",
	}
	for _, tc := range cases {
		_, err := NewEmail(tc)
		if err == nil {
			t.Errorf("NewEmail(%q) expected error, got nil", tc)
		}
	}
}

func TestRole_IsValid(t *testing.T) {
	if !RoleClient.IsValid() {
		t.Error("RoleClient should be valid")
	}
	if !RoleCompanion.IsValid() {
		t.Error("RoleCompanion should be valid")
	}
	if !RoleAdmin.IsValid() {
		t.Error("RoleAdmin should be valid")
	}
	if Role("UNKNOWN").IsValid() {
		t.Error("UNKNOWN role should be invalid")
	}
}

func TestRole_CanUpgradeToCompanion(t *testing.T) {
	if !RoleClient.CanUpgradeToCompanion() {
		t.Error("CLIENT should be able to upgrade")
	}
	if RoleCompanion.CanUpgradeToCompanion() {
		t.Error("COMPANION should NOT be able to upgrade again")
	}
	if RoleAdmin.CanUpgradeToCompanion() {
		t.Error("ADMIN should NOT be able to upgrade")
	}
}

func TestAccountStatus_Transitions(t *testing.T) {
	if !StatusActive.CanLogin() {
		t.Error("ACTIVE should allow login")
	}
	if StatusLocked.CanLogin() {
		t.Error("LOCKED should NOT allow login [INV-ID01]")
	}
	if !StatusActive.CanLock() {
		t.Error("ACTIVE should be lockable")
	}
	if StatusLocked.CanLock() {
		t.Error("LOCKED should NOT be lockable again")
	}
	if !StatusLocked.CanUnlock() {
		t.Error("LOCKED should be unlockable")
	}
	if StatusActive.CanUnlock() {
		t.Error("ACTIVE should NOT be unlockable")
	}
}

func TestUpgradeStatus_Transitions(t *testing.T) {
	if !UpgradeStatusPending.CanApprove() {
		t.Error("PENDING should be approvable")
	}
	if UpgradeStatusApproved.CanApprove() {
		t.Error("APPROVED should NOT be approvable")
	}
	if !UpgradeStatusPending.CanReject() {
		t.Error("PENDING should be rejectable")
	}
	if UpgradeStatusRejected.CanReject() {
		t.Error("REJECTED should NOT be rejectable")
	}
}

func TestUserID_ParseAndEquals(t *testing.T) {
	id1 := NewUserID()
	id2, err := ParseUserID(id1.String())
	if err != nil {
		t.Fatalf("ParseUserID failed: %v", err)
	}
	if !id1.Equals(id2) {
		t.Error("parsed ID should equal original")
	}

	_, err = ParseUserID("invalid-uuid")
	if err == nil {
		t.Error("ParseUserID should fail for invalid UUID")
	}
}

func TestParseRole_Valid(t *testing.T) {
	r, err := ParseRole("CLIENT")
	if err != nil {
		t.Fatalf("ParseRole(CLIENT) failed: %v", err)
	}
	if r != RoleClient {
		t.Errorf("expected CLIENT, got %s", r)
	}
}

func TestParseRole_Invalid(t *testing.T) {
	_, err := ParseRole("UNKNOWN")
	if err == nil {
		t.Error("ParseRole(UNKNOWN) should fail")
	}
}
