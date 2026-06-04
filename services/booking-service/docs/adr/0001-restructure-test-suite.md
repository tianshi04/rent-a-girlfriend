# ADR 0001: Tái cấu trúc bộ kiểm thử (Test Suite) của Booking Service thành các thư mục chuyên biệt

**Trạng thái:** Accepted  
**Ngày:** 2026-06-04  

---

## Ngữ cảnh (Context)

Bộ kiểm thử của dịch vụ **Booking Service** trước đây được đặt chung tại thư mục gốc `tests` bao gồm cả Contract, Integration, E2E và SAGA tests. Điều này dẫn đến một số vấn đề:
1.  **Thiếu nhất quán với các dịch vụ khác:** Các microservices khác (như `identity-service`) đã tổ chức các bài kiểm thử thành các nhóm chuyên biệt (`contract`, `e2e`, `integration`).
2.  **Khó quản lý và mở rộng:** Tệp E2E test quá lớn chứa tất cả các ca kiểm thử cho nhiều endpoint, gây khó khăn cho việc định vị lỗi và mở rộng các ca kiểm thử mới.
3.  **Lẫn lộn giữa kịch bản kiểm thử và các công cụ giả lập thủ công (mock scripts):** Thư mục `scripts` chứa các tệp như `publish_mock_event.go` và `mock_grpc/main.go` dùng cho việc chạy thử thủ công nhưng chưa được phân loại vào đúng phân vùng kiểm thử (tests).

Do đó, cần có một phương án phân tách cấu trúc tệp tin kiểm thử rõ ràng để dễ bảo trì và đồng bộ hóa kiến trúc kiểm thử toàn dự án.

---

## Quyết định (Decision)

Chúng tôi quyết định tái cấu trúc toàn bộ tệp tin kiểm thử của `booking-service` theo các chuẩn hóa sau:

### 1. Phân chia thư mục kiểm thử:
Di chuyển tất cả các tệp kiểm thử từ thư mục gốc `tests` vào các thư mục con:
*   `tests/contract/`: Chứa các bài kiểm thử tuần tự hóa dữ liệu (Protobuf, JSON) và tính tương thích enum.
*   `tests/integration/`: Chứa các bài kiểm thử tích hợp thực tế với PostgreSQL (qua GORM) và điều phối giao dịch phân tán SAGA (qua Kafka).
*   `tests/e2e/`: Chứa các bài kiểm thử HTTP/gRPC gateway endpoint sử dụng in-memory mock.
*   `tests/mock/mock_event/`: Chứa tệp giả lập đẩy event vào Kafka (`publish_mock_event.go`).
*   `tests/mock/mock_grpc/`: Chứa server giả lập gRPC (`mock_grpc.go`).
*Lưu ý: Việc tách các mock script chạy thủ công vào các thư mục con riêng biệt để tránh lỗi trùng lặp khai báo hàm main của trình biên dịch Go.*

### 2. Mô-đun hóa E2E Tests:
*   Chia nhỏ tệp tin `tests/e2e_test.go` khổng lồ thành tệp tin `testutil.go` (định nghĩa mocks chung và helper start server) và các tệp tin test cụ thể cho từng nghiệp vụ (`get_booking_test.go`, `list_bookings_test.go`, `request_booking_test.go`, v.v.).

### 3. Đồng bộ hóa Makefile và Test Runners:
*   Cập nhật `Makefile` tách biệt các câu lệnh chạy test tương ứng: `make test-unit`, `make test-contract`, `make test-e2e`, `make test-integration`, và `make test-all`.
*   Viết lại script chạy test tích hợp `run_integration_tests.sh` (chỉ nhắm vào `./tests/integration/...`).

---

## Hệ quả (Consequences)

### Điểm tích cực (Positives):
*   **Đồng bộ cấu trúc toàn dự án:** Đưa cấu trúc kiểm thử của `booking-service` về cùng một khuôn mẫu với `identity-service`, tạo sự nhất quán cho các nhà phát triển.
*   **Dễ bảo trì và mở rộng:** Các file test E2E nhỏ gọn, tập trung vào một nhóm API nghiệp vụ duy nhất.
*   **Làm sạch thư mục Scripts:** Thư mục `scripts` chỉ còn lại các file script tự động hóa thực tế (`gen_proto.sh`, `run_integration_tests.sh`), các mock được chuyển về vùng `tests/mock/`.

### Đánh đổi (Negatives):
*   Các nhà phát triển cần làm quen với các lệnh kiểm thử chi tiết hơn thay vì chỉ sử dụng một lệnh `make test` hoặc `make test-e2e` gộp chung như trước.
