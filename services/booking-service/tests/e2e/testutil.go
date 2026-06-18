package e2e

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

// getBaseURL trả về base URL của Booking Service HTTP API.
func getBaseURL() string {
	if baseURL := os.Getenv("E2E_BASE_URL"); baseURL != "" {
		return baseURL
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081" // Trùng với cổng đã map trong docker-compose.test.yml
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// getTestDB khởi tạo kết nối GORM PostgreSQL dùng cho E2E test.
func getTestDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/booking_test_db?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Đảm bảo các bảng cơ sở dữ liệu đã tồn tại
	err = db.AutoMigrate(
		&persistence.BookingModel{},
		&persistence.OutboxModel{},
		&persistence.BookingAcceptSagaModel{},
		&persistence.ProcessedEventModel{},
	)
	if err != nil {
		t.Fatalf("failed to run GORM auto-migrations: %v", err)
	}

	return db
}

// truncateTables dọn dẹp dữ liệu các bảng trước/sau mỗi bài test.
func truncateTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE bookings CASCADE")
	db.Exec("TRUNCATE TABLE outbox CASCADE")
	db.Exec("TRUNCATE TABLE booking_accept_sagas CASCADE")
	db.Exec("TRUNCATE TABLE processed_events CASCADE")
}

// TestMain chạy trước tất cả các test trong package e2e để chờ ứng dụng sẵn sàng.
func TestMain(m *testing.M) {
	waitForService()
	os.Exit(m.Run())
}

// waitForService thăm dò endpoint /health/ready cho đến khi ứng dụng khởi chạy thành công.
func waitForService() {
	healthURL := fmt.Sprintf("%s/health/ready", getBaseURL())
	deadline := time.Now().Add(60 * time.Second)

	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			fmt.Println("[testutil] Booking Service is healthy, starting E2E tests...")
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		fmt.Printf("[testutil] Waiting for service at %s...\n", healthURL)
		time.Sleep(2 * time.Second)
	}

	fmt.Fprintf(os.Stderr, "[testutil] FATAL: service did not become healthy within 60s at %s\n", healthURL)
	os.Exit(1)
}
