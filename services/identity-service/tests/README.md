# Hướng Dẫn Chạy Test Identity Service

Thư mục này chứa toàn bộ các bài kiểm thử (Integration, E2E, Contract) của Identity Service. Dưới đây là danh sách tổng hợp tất cả các câu lệnh cần thiết để thực thi từng loại test.

## 1. Unit Tests
Unit Test kiểm thử các logic cốt lõi (Domain/Application) mà **không cần** bất kỳ phụ thuộc bên ngoài nào (như Database, Kafka, Redis). Các phụ thuộc đã được giả lập (Mock).

**Lệnh chạy:**
```bash
# Chạy tất cả Unit Tests (loại bỏ các bài test tích hợp/E2E mất thời gian)
go test -short -v ./...

# Hoặc chỉ chạy các test bên trong thư mục internal
go test -v ./internal/...

# Chạy test và xem coverage
go test -short -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 2. Integration Tests
Integration Test nằm trong thư mục `tests/integration/` nhằm kiểm tra sự kết nối giữa mã nguồn và các hệ thống thật (PostgreSQL, Redis, Kafka). 
> **Yêu cầu:** Phải khởi động Docker Compose chứa Database, Redis và Kafka ở máy local trước.

**Lệnh chạy:**
```bash
# Di chuyển về thư mục gốc của Identity Service
cd ../

# Chạy tất cả Integration Tests
go test -v ./tests/integration/...

# Chạy một file Integration Test cụ thể
go test -v ./tests/integration/db_test.go
```

## 3. End-to-End (E2E) Tests
E2E Test nằm trong thư mục `tests/e2e/`. Hệ thống E2E của chúng ta đã được tự động hóa **100% bằng Docker Compose** để cung cấp môi trường hoàn toàn cô lập (Postgres riêng, Redis riêng, Identity Service đóng gói).

> **Tuyệt vời:** Bạn KHÔNG CẦN cài đặt Golang, KHÔNG CẦN bật database thủ công, chỉ cần có Docker Desktop là đủ!

**Chỉ cần chạy 1 lệnh duy nhất (từ thư mục `services/identity-service`):**

**Nếu bạn dùng Make (Linux/Mac/WSL):**
```bash
make test-e2e
```

**Nếu bạn dùng Bash Script (Linux/Mac/WSL/Git Bash):**
```bash
bash scripts/run_e2e.sh
```

**Nếu bạn dùng PowerShell (Windows):**
```powershell
.\scripts\run_e2e.ps1
```

*Lệnh trên sẽ tự động Build Docker Image, khởi chạy môi trường Test DB/Redis, gọi bộ Test Suite bằng `e2e-runner`, và tự động xóa sạch rác (cleanup) sau khi hoàn thành!*

---

## 4. Chạy Toàn Bộ Test Cơ Bản Cùng Lúc
Nếu bạn đã chuẩn bị đầy đủ môi trường (Docker Database, Redis, Kafka đang chạy) và muốn chạy **toàn bộ** các test (Unit + Integration), bạn có thể gán các biến môi trường cấu hình và chạy:

**Windows (PowerShell):**
```powershell
$env:ENABLE_TEST_ROUTES="true"
$env:APP_ENV="test"
$env:DATABASE_URL="postgres://test:test@localhost:5433/identity_test?sslmode=disable"
$env:REDIS_URL="redis://localhost:6380"

go test -v ./...
```

**Linux / Mac:**
```bash
ENABLE_TEST_ROUTES=true APP_ENV=test DATABASE_URL="postgres://test:test@localhost:5433/identity_test?sslmode=disable" REDIS_URL="redis://localhost:6380" go test -v ./...
```
