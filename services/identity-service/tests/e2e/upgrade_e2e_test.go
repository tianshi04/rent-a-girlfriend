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

func TestCompanionUpgradeE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// 1. Create a Client user and get token
	clientEmail := fmt.Sprintf("client_%d@example.com", time.Now().UnixNano())
	clientGoogleID := fmt.Sprintf("client-google-id-%d", time.Now().UnixNano())
	clientAccessToken, _, _ := mockLogin(t, clientEmail, clientGoogleID, "CLIENT")

	// 2. Client requests upgrade
	reqURL := fmt.Sprintf("%s/api/v1/upgrade-requests", getBaseURL())
	reqPayload := map[string]string{
		"reason": "I want to be a companion",
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+clientAccessToken)

	clientHTTP := &http.Client{Timeout: 5 * time.Second}
	resp, err := clientHTTP.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// 3. Create an Admin user and get token
	adminEmail := fmt.Sprintf("admin_%d@example.com", time.Now().UnixNano())
	adminGoogleID := fmt.Sprintf("admin-google-id-%d", time.Now().UnixNano())
	adminAccessToken, _, _ := mockLogin(t, adminEmail, adminGoogleID, "ADMIN")

	// 4. Admin lists upgrade requests
	listURL := fmt.Sprintf("%s/api/v1/admin/upgrade-requests?status=PENDING", getBaseURL())
	listReq, _ := http.NewRequest(http.MethodGet, listURL, nil)
	listReq.Header.Set("Authorization", "Bearer "+adminAccessToken)

	listResp, err := clientHTTP.Do(listReq)
	assert.NoError(t, err)
	defer listResp.Body.Close()

	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var listResult struct {
		Data []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	err = json.NewDecoder(listResp.Body).Decode(&listResult)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResult.Data), 1)

	// Get the first pending request ID
	requestID := listResult.Data[0].ID

	// 5. Admin approves the request
	approveURL := fmt.Sprintf("%s/api/v1/admin/upgrade-requests/%s/approve", getBaseURL(), requestID)
	approveReq, _ := http.NewRequest(http.MethodPut, approveURL, nil)
	approveReq.Header.Set("Authorization", "Bearer "+adminAccessToken)

	approveResp, err := clientHTTP.Do(approveReq)
	assert.NoError(t, err)
	defer approveResp.Body.Close()

	assert.Equal(t, http.StatusOK, approveResp.StatusCode)
}
