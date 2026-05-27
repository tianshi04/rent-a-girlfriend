# ADR 0002: Triển khai Graceful Shutdown cho Interaction Service

**Trạng thái:** Accepted
**Ngày:** 2026-05-27

## Ngữ cảnh (Context)
Dịch vụ `interaction-service` (viết bằng Rust) chạy song song nhiều cấu phần:
1. Axum HTTP REST Server (cổng `8080`).
2. Tonic gRPC Server (cổng `50051`).
3. OutboxWorker (quét bảng `outbox` Postgres và đẩy lên Kafka).
4. BookingEventListener (lắng nghe các sự kiện Kafka).

Hiện tại, việc dừng ứng dụng chỉ bắt tín hiệu `ctrl_c` (SIGINT) và dừng main thread đột ngột bằng `tokio::select!`. Cơ chế này mang lại các rủi ro:
* Không bắt được tín hiệu `SIGTERM` từ Kubernetes dẫn đến việc bị kill cứng không báo trước.
* Các request in-flight đang xử lý qua HTTP/gRPC bị ngắt kết nối đột ngột, gây lỗi cho Clients hoặc các services khác gọi sang.
* Các background worker bị dừng đột ngột giữa chu kỳ xử lý, dẫn đến rủi ro sai lệch trạng thái Kafka offset hoặc transaction Postgres.

## Quyết định (Decision)
Triển khai cơ chế Graceful Shutdown toàn diện và tập trung:
1. **Lắng nghe Tín hiệu Đa nền tảng:** Sử dụng `tokio::signal::ctrl_c` và `tokio::signal::unix::signal` để bắt cả `SIGINT` (Ctrl+C local) và `SIGTERM` (Kubernetes production).
2. **Cơ chế Phát tín hiệu Tập trung (Broadcasting):** Sử dụng một `tokio::sync::watch` channel phát tín hiệu `shutdown = true` tới tất cả các server và background worker.
3. **Graceful Server Integration:**
   - Cấu hình Axum HTTP Server sử dụng `.with_graceful_shutdown()`.
   - Cấu hình Tonic gRPC Server sử dụng `.serve_with_shutdown()`.
4. **Graceful Background Workers:**
   - Cung cấp `shutdown_rx: Receiver<bool>` của watch channel cho các background worker.
   - Sử dụng `tokio::select!` trong vòng lặp vô hạn của worker để thoát loop an toàn và nhanh chóng khi nhận tín hiệu shutdown mà không bị block tại các lệnh `sleep` hoặc Kafka consumer `.recv()`.
5. **Timeout Guard bảo vệ:** Khi kích hoạt shutdown, main thread sẽ đợi toàn bộ các tác vụ tắt xong. Bổ sung một `tokio::time::sleep` timeout guard (15 giây) để ép buộc thoát nếu có tác vụ bị treo vĩnh viễn (do lỗi kết nối hạ tầng) để tránh treo pod Kubernetes vô hạn.

## Hệ quả (Consequences)
### Điểm tích cực (Positives)
* **An toàn dữ liệu:** Đảm bảo các request HTTP/gRPC đang chạy được xử lý trọn vẹn trước khi kết nối đóng hẳn.
* **Đồng bộ Kafka:** Giảm thiểu rủi ro xử lý trùng lặp sự kiện do consumer thoát đột ngột chưa kịp commit offset.
* **Kubernetes Native:** Tương thích hoàn hảo với chu trình tắt pod của Kubernetes.
* **Không tốn thêm dependency:** Sử dụng các tính năng sẵn có của `tokio` (watch channel, select, join) giúp mã nguồn nhẹ và dễ bảo trì.

### Đánh đổi (Negatives)
* **Thời gian shutdown kéo dài:** Pod sẽ mất vài giây đến tối đa 15 giây để tắt hoàn toàn thay vì tắt ngay lập tức. Cần thiết lập `terminationGracePeriodSeconds` tối thiểu 30 giây trong Kubernetes deployment sau này.
