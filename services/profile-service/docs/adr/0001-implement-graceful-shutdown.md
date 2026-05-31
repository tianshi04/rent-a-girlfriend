# ADR 0001: Implement Coordinated Graceful Shutdown for Python Microservice

**Trạng thái:** Accepted

**Ngày:** 2026-05-31

## Ngữ cảnh (Context)

Dịch vụ `profile-service` là một microservice viết bằng Python, chạy đồng thời ba thành phần bất đồng bộ (asyncio):
1. **Uvicorn HTTP/REST Server** (FastAPI) để phục vụ các yêu cầu truy vấn (REST queries).
2. **gRPC Server** để phục vụ các lệnh nghiệp vụ đồng bộ (Commands).
3. **Outbox Worker** chạy nền để gửi các Domain Events từ bảng Outbox sang Kafka (At-Least-Once Delivery).

Trước đây, cơ chế tắt (shutdown) của dịch vụ gặp phải các vấn đề nghiêm trọng:
- Không lắng nghe tín hiệu `SIGTERM` (tín hiệu tiêu chuẩn do Kubernetes/Docker gửi để dừng container).
- Khi bắt được `KeyboardInterrupt` (`SIGINT`), ứng dụng thoát khỏi event loop chính và khởi tạo lại một event loop mới thông qua `asyncio.run(outbox_worker.stop())` để dừng worker. Điều này gây lỗi runtime vì loop cũ đã đóng và loop mới không quản lý đúng tài nguyên đang chạy.
- Uvicorn và gRPC server không được dừng một cách chủ động và an toàn (gracefully), dẫn đến việc các kết nối/yêu cầu đang xử lý bị ngắt đột ngột khi container bị force-kill (`SIGKILL`).

Do đó, cần có một cơ chế lifecycle quản lý tập trung và phối hợp nhịp nhàng việc khởi tạo cũng như tắt toàn bộ các thành phần trên một event loop duy nhất.

## Quyết định (Decision)

Chúng tôi quyết định chuẩn hóa cơ chế lifecycle quản lý vòng đời chạy của ứng dụng Python theo các nguyên tắc sau:
1. **Một Event Loop Duy Nhất**: Toàn bộ các máy chủ và tiến trình nền (Uvicorn, gRPC, Outbox Worker) phải chạy đồng thời dưới dạng các `asyncio.Task` trong cùng một event loop khởi tạo bởi `asyncio.run(main())`.
2. **Quản lý Tín Hiệu Tập Trung**:
   - Vô hiệu hóa tính năng tự động đăng ký bộ xử lý tín hiệu của Uvicorn bằng cách đặt `install_signal_handlers=False` trong cấu hình Uvicorn.
   - Sử dụng `loop.add_signal_handler` để lắng nghe cả hai tín hiệu `SIGINT` và `SIGTERM` ở cấp độ event loop của ứng dụng, với giải pháp fallback an toàn cho hệ điều hành Windows (`NotImplementedError`).
   - Sử dụng `asyncio.Event` (`shutdown_event`) để thông báo yêu cầu tắt máy cho toàn bộ ứng dụng.
3. **Quy trình Shutdown Tuần tự và Đồng bộ (Coordinated Graceful Shutdown)**:
   - Toàn bộ chuỗi shutdown được đặt trong khối `finally` của hàm `main()`. Điều này đảm bảo rằng dù ứng dụng dừng do nhận tín hiệu, do có lỗi phát sinh bên trong (exception), hoặc do tác vụ nền bị hủy (`asyncio.CancelledError`), chuỗi shutdown vẫn được thực thi đầy đủ và an toàn.
   - Thứ tự dừng cụ thể:
     1. **HTTP/REST Server (Uvicorn)**: Đặt `http_server.should_exit = True` và đợi tác vụ hoàn thành (timeout 5 giây).
     2. **gRPC Server**: Gọi `await grpc_server.stop(grace=5.0)` và đợi kết thúc các RPC đang hoạt động.
     3. **Outbox Worker**: Gọi `await outbox_worker.stop()` để dừng polling loop và đóng Kafka producer an toàn.

## Hệ quả (Consequences)

### Điểm tích cực (Positives)
- **Độ tin cậy cao**: Toàn bộ kết nối HTTP hoạt động và các tiến trình gRPC RPC đang xử lý được hoàn thành an toàn trước khi dừng hẳn.
- **Tính toàn vẹn dữ liệu**: Outbox worker dừng polling và kết thúc việc gửi các lô event dở dang sang Kafka, giảm thiểu nguy cơ trùng lặp hoặc mất event.
- **Tránh rác tài nguyên**: Loại bỏ hoàn toàn lỗi tạo mới event loop trong quá trình cleanup, đảm bảo tài nguyên hệ điều hành (sockets, connection pools) được giải phóng sạch sẽ.
- **Chuẩn hóa container**: Phản hồi tốt với cả `SIGINT` và `SIGTERM`, tương thích hoàn hảo với Kubernetes/Docker lifecycle.

### Đánh đổi (Negatives)
- **Tăng độ phức tạp của code khởi động**: Hàm `main()` cần quản lý các `asyncio.Task` chạy nền của các server thay vì chỉ gọi `asyncio.gather` đơn giản như trước.
- **Phụ thuộc vào thời gian Grace Period**: Việc tắt cần một khoảng thời gian chờ (grace period) tối đa 5 giây cho mỗi dịch vụ trước khi buộc phải dừng hẳn.
