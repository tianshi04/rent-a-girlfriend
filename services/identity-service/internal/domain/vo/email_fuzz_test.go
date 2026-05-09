package vo

import (
	"testing"
)

func FuzzNewEmail(f *testing.F) {
	// Add some seed corpus
	f.Add("user@example.com")
	f.Add("invalid-email")
	f.Add("")
	f.Add("   ")
	f.Add(string([]byte{0, 1, 2, 3}))

	f.Fuzz(func(t *testing.T, emailStr string) {
		email, err := NewEmail(emailStr)
		if err == nil {
			// If it's valid, it should be possible to reconstruct it
			reconstructed, err := NewEmail(email.String())
			if err != nil {
				t.Errorf("failed to reconstruct valid email %q: %v", email.String(), err)
			}
			if reconstructed.String() != email.String() {
				t.Errorf("reconstructed email mismatch: %q != %q", reconstructed.String(), email.String())
			}
		}
	})
}
