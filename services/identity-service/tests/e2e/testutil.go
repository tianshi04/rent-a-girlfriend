package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

// getBaseURL returns the base URL of the Identity Service HTTP API
func getBaseURL() string {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081" // Default port where Docker exposes the service
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// Helper to make the Mock Login HTTP call
func mockLogin(t *testing.T, email, googleID, role string) (string, string, string) { // returns accessToken, refreshToken, userID
	url := fmt.Sprintf("%s/api/v1/test/mock-login", getBaseURL())

	payload := map[string]string{
		"email":     email,
		"google_id": googleID,
		"role":      role,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to call mock login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Mock login failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode mock login response: %v", err)
	}

	accessToken, ok1 := result["accessToken"].(string)
	refreshToken, ok2 := result["refreshToken"].(string)

	if !ok1 || !ok2 {
		t.Fatalf("Mock login response missing tokens")
	}

	// Extract userID from JWT claims (sub)
	userID := extractUserIDFromToken(t, accessToken)

	return accessToken, refreshToken, userID
}

func extractUserIDFromToken(t *testing.T, token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("Invalid JWT token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("Failed to decode JWT payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("Failed to unmarshal JWT claims: %v", err)
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		t.Fatalf("JWT claims missing 'sub' field")
	}

	return sub
}
