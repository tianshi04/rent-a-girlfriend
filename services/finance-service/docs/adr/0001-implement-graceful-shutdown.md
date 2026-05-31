# ADR 0001: Triển khai Graceful Shutdown cho Finance Service

**Trạng thái:** Accepted
**Ngày:** 2026-05-31

## Ngữ cảnh (Context)
Dịch vụ `finance-service` (viết bằng Python) chạy song song nhiều cấu phần:
1. FastAPI / Uvicorn HTTP Server (cổng `8080`).
2. gRPC Server (cổng `50051`).
3. OutboxPublisherWorker (quét bảng `outbox` Postgres và đẩy lên Kafka).
4. Identity Event Listener (lắng nghe các sự kiện Kafka để onboard ví).

Hiện tại, việc dừng ứng dụng chỉ bắt tín hiệu KeyboardInterrupt (Ctrl+C) ở main entrypoint, hoàn toàn bỏ qua `SIGTERM` từ Kubernetes. Thêm vào đó:
* Khi nhận được KeyboardInterrupt, `asyncio.run(outbox_worker.stop())` được gọi ngoài event loop ban đầu đã đóng. Điều này tạo ra một event loop mới và không thể dọn dẹp tài nguyên gán với loop đã bị hủy, dẫn đến các ngoại lệ lỗi kết nối hoặc loop closed.
* gRPC server và Uvicorn HTTP server không bao giờ được dừng một cách graceful. Phương thức `server.stop(grace=...)` của gRPC server không được gọi.
* Background Kafka listener (`run_identity_event_listener`) bị hủy nhưng các timeout close của consumer không được quản lý sạch sẽ.
* Outbox workers không được dừng lại khi nhận `SIGTERM`.

## Quyết định (Decision)
Triển khai cơ chế Graceful Shutdown tập trung và an toàn:
1. **Lắng nghe Tín hiệu Đa nền tảng:** Bắt cả tín hiệu `SIGINT` (Ctrl+C) và `SIGTERM` (Kubernetes default shutdown signal). Sử dụng `loop.add_signal_handler` cho môi trường tương thích POSIX/Linux và fallback an toàn sang `signal.signal` kết hợp `loop.call_soon_threadsafe` trên Windows.
2. **Quản lý Vòng đời Tập trung:** Định nghĩa `shutdown_event = asyncio.Event()` để báo hiệu tắt toàn bộ hệ thống.
3. **Graceful Server Shutdown:**
   - Với gRPC Server, gọi `await grpc_server.stop(grace=5)` khi nhận tín hiệu shutdown để dừng nhận yêu cầu mới và chờ các yêu cầu hiện tại xử lý xong.
   - Với Uvicorn Server, thiết lập `http_server.should_exit = True` để kích hoạt shutdown chu trình của Uvicorn một cách sạch sẽ.
4. **Dọn dẹp Worker và Consumer trong cùng Event Loop:**
   - Hủy task Identity Event Listener và await hoàn tất để đảm bảo block `finally` của nó chạy `await consumer.stop()`.
   - Dừng `outbox_worker` bằng cách gọi `await outbox_worker.stop()` ngay trong cùng event loop trước khi tiến trình kết thúc.
5. **Giới hạn Thời gian Dọn dẹp (Timeout Guard):** Sử dụng `asyncio.wait_for` giới hạn tối đa 5 giây cho việc dừng các server. Nếu vượt quá thời gian, hủy cứng các tác vụ để tránh treo pod Kubernetes vô hạn.

## Hệ quả (Consequences)
### Điểm tích cực (Positives)
* **Tính toàn vẹn dữ liệu:** Các giao dịch tài chính đang xử lý dở dang được đảm bảo kết thúc an toàn, tránh lỗi mất kết nối nửa chừng.
* **Ngăn chặn Lỗi Event Loop:** Tránh triệt để các lỗi `RuntimeError: Event loop is closed` do thao tác dừng Kafka/outbox ngoài loop chính.
* **Tương thích Tốt:** Đảm bảo hoạt động trơn tru trong Kubernetes khi pod bị tắt/tái khởi động.

### Đánh đổi (Negatives)
* Tiến trình tắt sẽ mất tối đa vài giây tùy thuộc vào các kết nối và tác vụ đang xử lý, nhưng nằm trong tầm kiểm soát nhờ timeout guard.
