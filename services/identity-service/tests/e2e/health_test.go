package e2e

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	// Skip if not in E2E mode
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081"
	}
	url := "http://localhost:" + port + "/health"

	// Create a client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("failed to call health check: %v. Is the server running?", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
