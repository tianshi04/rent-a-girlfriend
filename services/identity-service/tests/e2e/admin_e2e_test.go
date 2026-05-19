package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAdminLockAccountE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// 1. Create a bad user
	badEmail := fmt.Sprintf("bad_%d@example.com", time.Now().UnixNano())
	badGoogleID := fmt.Sprintf("bad-google-id-%d", time.Now().UnixNano())
	badAccessToken, _, badUserID := mockLogin(t, badEmail, badGoogleID, "CLIENT")

	// Verify the bad user can access a protected route (e.g., Request Upgrade)
	reqURL := fmt.Sprintf("%s/api/v1/upgrade-requests", getBaseURL())
	reqPayload := map[string]string{
		"reason": "Test access",
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	injectMeshHeaders(t, req, badAccessToken)

	clientHTTP := &http.Client{Timeout: 30 * time.Second}
	resp, err := clientHTTP.Do(req)
	assert.NoError(t, err)
	if resp != nil {
		defer resp.Body.Close()
	}
	
	// They shouldn't be blocked (yet)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 2. Create Admin
	adminEmail := fmt.Sprintf("admin_lock_%d@example.com", time.Now().UnixNano())
	adminGoogleID := fmt.Sprintf("admin-lock-id-%d", time.Now().UnixNano())
	adminAccessToken, _, _ := mockLogin(t, adminEmail, adminGoogleID, "ADMIN")

	assert.NotEmpty(t, badUserID, "Could not find bad user ID to lock")

	// 3. Admin locks the account
	lockURL := fmt.Sprintf("%s/api/v1/admin/accounts/%s/lock", getBaseURL(), badUserID)
	lockPayload := map[string]string{
		"reason": "Spamming",
	}
	lockBody, _ := json.Marshal(lockPayload)

	lockReq, _ := http.NewRequest(http.MethodPut, lockURL, bytes.NewBuffer(lockBody))
	lockReq.Header.Set("Content-Type", "application/json")
	injectMeshHeaders(t, lockReq, adminAccessToken)

	lockResp, err := clientHTTP.Do(lockReq)
	assert.NoError(t, err)
	if lockResp != nil {
		defer lockResp.Body.Close()
	}

	if lockResp != nil && lockResp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		json.NewDecoder(lockResp.Body).Decode(&errBody)
		t.Logf("Lock failed error: %v", errBody)
	}
	assert.Equal(t, http.StatusOK, lockResp.StatusCode)

	// 4. Bad user tries to request upgrade again (Should fail with 403 Forbidden due to locked status)
	req2, _ := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	injectMeshHeaders(t, req2, badAccessToken)
	req2.Header.Set("user-status", "LOCKED") // Simulate service mesh propagating the updated status

	resp2, err := clientHTTP.Do(req2)
	assert.NoError(t, err)
	if resp2 != nil {
		defer resp2.Body.Close()
	}

	assert.Equal(t, http.StatusForbidden, resp2.StatusCode)
	t.Logf("Status after lock: %d", resp2.StatusCode)
}
