package e2e

import (
	"net/http"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	// Skip if not in E2E mode
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	endpoints := []string{"/health", "/health/live", "/health/ready"}
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			url := getBaseURL() + ep
			resp, err := client.Get(url)
			if err != nil {
				t.Fatalf("failed to call health check %s: %v", ep, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}
