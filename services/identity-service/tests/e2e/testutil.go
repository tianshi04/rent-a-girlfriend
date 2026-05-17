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
	"time"
)

// getBaseURL trả về base URL của Identity Service HTTP API.
// Có thể cấu hình qua E2E_BASE_URL, mặc định là http://localhost:8081.
func getBaseURL() string {
	if baseURL := os.Getenv("E2E_BASE_URL"); baseURL != "" {
		return baseURL
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// TestMain chạy trước tất cả test trong package e2e:
//  1. Đợi service sẵn sàng (health check polling)
//  2. TRUNCATE tất cả tables — đảm bảo state sạch trước mỗi suite
//  3. Chạy tests
//  4. (Cleanup được thực hiện bởi `make test-e2e` qua docker-compose down -v)
func TestMain(m *testing.M) {
	waitForService()
	truncateAllTables()
	os.Exit(m.Run())
}

// waitForService poll /health cho đến khi service trả về 200 hoặc timeout.
func waitForService() {
	healthURL := fmt.Sprintf("%s/health", getBaseURL())
	deadline := time.Now().Add(60 * time.Second)

	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Println("[testutil] Service is healthy, starting E2E tests...")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Printf("[testutil] Waiting for service at %s...\n", healthURL)
		time.Sleep(2 * time.Second)
	}

	fmt.Fprintf(os.Stderr, "[testutil] FATAL: service did not become healthy within 60s at %s\n", healthURL)
	os.Exit(1)
}

// truncateAllTables xóa sạch tất cả bảng dữ liệu user qua endpoint test.
// Service phải chạy với ENABLE_TEST_ROUTES=true.
func truncateAllTables() {
	truncateURL := fmt.Sprintf("%s/api/v1/test/truncate", getBaseURL())
	resp, err := http.Post(truncateURL, "application/json", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[testutil] WARNING: failed to call truncate endpoint: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		fmt.Fprintf(os.Stderr, "[testutil] WARNING: truncate returned status %d\n", resp.StatusCode)
		return
	}
	fmt.Println("[testutil] All tables truncated successfully")
}

// mockLogin gọi endpoint test để tạo user và nhận token mà không qua OAuth.
// Chỉ hoạt động khi ENABLE_TEST_ROUTES=true.
func mockLogin(t *testing.T, email, googleID, role string) (accessToken, refreshToken, userID string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/test/mock-login", getBaseURL())

	payload := map[string]string{
		"email":     email,
		"google_id": googleID,
		"role":      role,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to call mock-login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mock-login returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode mock-login response: %v", err)
	}

	accessToken, ok1 := result["accessToken"].(string)
	refreshToken, ok2 := result["refreshToken"].(string)
	if !ok1 || !ok2 {
		t.Fatalf("mock-login response missing tokens: %v", result)
	}

	userID = extractSubFromJWT(t, accessToken)
	return accessToken, refreshToken, userID
}

// extractSubFromJWT lấy claim "sub" từ JWT payload mà không verify signature.
func extractSubFromJWT(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode JWT payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("failed to unmarshal JWT claims: %v", err)
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		t.Fatalf("JWT claims missing 'sub' field: %v", claims)
	}
	return sub
}
