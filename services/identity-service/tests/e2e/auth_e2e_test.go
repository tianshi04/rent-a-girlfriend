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

func TestAuthE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	email := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	googleID := fmt.Sprintf("google-id-%d", time.Now().UnixNano())

	// 1. Mock Login (Simulates successful OAuth exchange)
	accessToken, refreshToken, _ := mockLogin(t, email, googleID, "CLIENT")
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// Sleep 1 second to ensure iat changes for deterministic accessTokens
	time.Sleep(1 * time.Second)

	// 2. Refresh Token
	refreshURL := fmt.Sprintf("%s/api/v1/auth/refresh", getBaseURL())
	refreshPayload := map[string]string{
		"refreshToken": refreshToken,
	}
	body, _ := json.Marshal(refreshPayload)
	
	resp, err := http.Post(refreshURL, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	newAccessToken := result["accessToken"].(string)
	newRefreshToken := result["refreshToken"].(string)

	assert.NotEmpty(t, newAccessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, accessToken, newAccessToken)
	assert.NotEqual(t, refreshToken, newRefreshToken)

	// 3. Logout using the new refresh token
	logoutURL := fmt.Sprintf("%s/api/v1/auth/logout", getBaseURL())
	logoutPayload := map[string]string{
		"refreshToken": newRefreshToken,
	}
	logoutBody, _ := json.Marshal(logoutPayload)

	logoutResp, err := http.Post(logoutURL, "application/json", bytes.NewBuffer(logoutBody))
	assert.NoError(t, err)
	defer logoutResp.Body.Close()

	assert.Equal(t, http.StatusOK, logoutResp.StatusCode)

	// 4. Try refreshing with the logged-out token (Should Fail)
	failRefreshResp, err := http.Post(refreshURL, "application/json", bytes.NewBuffer(logoutBody))
	assert.NoError(t, err)
	defer failRefreshResp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, failRefreshResp.StatusCode)
}
