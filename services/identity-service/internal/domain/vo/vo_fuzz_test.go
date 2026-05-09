package vo

import (
	"testing"
)

func FuzzParseUserID(f *testing.F) {
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("invalid")
	f.Add("")
	
	f.Fuzz(func(t *testing.T, idStr string) {
		id, err := ParseUserID(idStr)
		if err == nil {
			if id.String() == "" {
				t.Error("parsed ID should not have empty string representation")
			}
			reparsed, err := ParseUserID(id.String())
			if err != nil {
				t.Errorf("failed to re-parse ID %q: %v", id.String(), err)
			}
			if !id.Equals(reparsed) {
				t.Errorf("re-parsed ID mismatch: %v != %v", id, reparsed)
			}
		}
	})
}

func FuzzParseRole(f *testing.F) {
	f.Add("CLIENT")
	f.Add("COMPANION")
	f.Add("ADMIN")
	f.Add("UNKNOWN")
	
	f.Fuzz(func(t *testing.T, roleStr string) {
		role, err := ParseRole(roleStr)
		if err == nil {
			if !role.IsValid() {
				t.Errorf("parsed role %q should be valid", role)
			}
		}
	})
}

func FuzzParseAccountStatus(f *testing.F) {
	f.Add("ACTIVE")
	f.Add("LOCKED")
	
	f.Fuzz(func(t *testing.T, statusStr string) {
		status, err := ParseAccountStatus(statusStr)
		if err == nil {
			if status != StatusActive && status != StatusLocked {
				t.Errorf("unexpected status parsed: %q", status)
			}
		}
	})
}

func FuzzParseUpgradeStatus(f *testing.F) {
	f.Add("PENDING")
	f.Add("APPROVED")
	f.Add("REJECTED")
	
	f.Fuzz(func(t *testing.T, statusStr string) {
		status, err := ParseUpgradeStatus(statusStr)
		if err == nil {
			// Ensure it matches one of the known statuses
			valid := false
			for _, s := range []UpgradeStatus{UpgradeStatusPending, UpgradeStatusApproved, UpgradeStatusRejected} {
				if status == s {
					valid = true
					break
				}
			}
			if !valid {
				t.Errorf("unexpected upgrade status parsed: %q", status)
			}
		}
	})
}
