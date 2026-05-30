# ADR 0009: Distributed SSE & Redis Pub/Sub Channel Strategy with Docker Integration

## Trạng thái (Status)
Đã duyệt (Accepted)

## Bối cảnh (Context)
Hệ thống **Notification Service** (Spring Boot) cần duy trì kết nối Server-Sent Events (SSE) thời gian thực với hàng ngàn người dùng trong môi trường phân tán (Distributed Environment - Nhiều Pods hoạt động đằng sau một Load Balancer). 

Khi có một sự kiện thông báo mới cho `userId`:
1. Sự kiện này được Kafka Consumer phân phối ngẫu nhiên cho một Pod bất kỳ (ví dụ: Pod C).
2. Client thực tế có thể đang duy trì kết nối HTTP/SSE mở trên một Pod khác (ví dụ: Pod A).
3. Pod C cần một cơ chế truyền thông điệp liên-máy (Inter-Pod Communication) thời gian thực để báo cho Pod A đẩy dữ liệu qua socket SSE xuống Client.

Chúng ta quyết định chọn **Redis Pub/Sub** làm cầu nối định tuyến thông tin thời gian thực nhờ tốc độ xử lý nhanh, độ trễ cực thấp. Tuy nhiên, chúng ta cần quyết định **Chiến lược phân kênh (Channel Strategy)** trên Redis Pub/Sub để tối ưu hóa hiệu năng, băng thông mạng và quản lý tài nguyên an toàn.

Để đảm bảo tính đồng nhất 100% (Parity) giữa môi trường Phát triển cục bộ (Local Dev) và Production, hạn chế tối đa các lỗi cấu hình hay hành vi khác biệt khi deploy, hệ thống sẽ sử dụng **Redis thật tự host trên Docker container cục bộ** ngay tại môi trường phát triển, thay vì dùng bất kỳ giải pháp giả lập In-Memory nào.

## Quyết định (Decision)

### 1. Chọn Chiến lược Kênh riêng biệt theo User (`user:{userId}:sse`)
Chúng ta chọn giải pháp **Kênh riêng biệt động** cho từng người dùng thay vì một kênh chung phát sóng (Broadcast Channel) duy nhất cho toàn hệ thống.
*   Mỗi khi Client kết nối tới Pod A, Pod A sẽ đăng ký lắng nghe (subscribe) vào kênh có tên `user:{userId}:sse` trên Redis.
*   Khi cần gửi thông báo, Pod C chỉ cần publish trực tiếp dữ liệu vào kênh `user:{userId}:sse`. Chỉ Pod A đang thực sự giữ kết nối của User mới nhận được tin nhắn và đẩy xuống TCP socket. Các Pod khác hoàn toàn không bị ảnh hưởng.
*   Thiết kế này loại bỏ triệt để hiện tượng **Thundering Herd** (tất cả các Pod nhận tin và phải parse dữ liệu rác để kiểm tra), tối ưu hóa băng thông truyền tin nội bộ trong mạng Kubernetes Cluster.

### 2. Áp dụng cơ chế Đếm kết nối (Reference Counting) để quản lý Subscription động
Để tránh overhead (chi phí) quá lớn do liên tục gửi các lệnh `SUBSCRIBE`/`UNSUBSCRIBE` lên Redis, đồng thời đảm bảo an toàn đa luồng (thread-safe):
*   Một User có thể mở nhiều kết nối active đồng thời (Ví dụ: 1 Web Frontend + 1 App Mobile cùng đăng nhập). Pod giữ kết nối sẽ quản lý danh sách `List<SseEmitter>` cục bộ cho User đó.
*   Chúng ta áp dụng cơ chế **Reference Counting (Đếm số kết nối)**:
    *   **Khi kết nối đầu tiên** của `userId` được tạo trên Pod đó (số lượng emitter tăng từ `0` lên `1`): Thực hiện gửi lệnh `SUBSCRIBE` tới kênh Redis `user:{userId}:sse`.
    *   **Khi có kết nối thứ N** được thêm vào (số lượng emitter > 1): Chỉ cần add `SseEmitter` vào danh sách cục bộ, **KHÔNG** gửi thêm lệnh subscribe lên Redis.
    *   **Khi một kết nối bị ngắt** (completion/timeout/error): Loại bỏ `SseEmitter` khỏi danh sách.
    *   **Khi kết nối cuối cùng** của User trên Pod đó bị ngắt (số lượng emitter giảm về `0`): Thực hiện gửi lệnh `UNSUBSCRIBE` tới Redis để giải phóng tài nguyên lắng nghe, tránh rò rỉ (memory leak) bộ đăng ký trên Redis Cluster.

### 3. Đồng nhất môi trường bằng Redis Docker cục bộ
*   Không xây dựng Adapter in-memory tạm thời. 
*   Bắt buộc tích hợp và chạy **Redis Container thực tế** (`redis:alpine`) bằng Docker/WSL trên máy của lập trình viên khi phát triển cục bộ.
*   Cả hai môi trường Local Dev và Production đều chạy chung duy nhất một triển khai Adapter: **`RedisPubSubAdapter`** (sử dụng Spring Data Redis `RedisMessageListenerContainer`).
*   **Kết quả:** Loại bỏ hoàn toàn sự khác biệt về mặt hành vi của môi trường (Environment Discrepancy), đảm bảo code được phát triển và chạy thử nghiệm ở local hoạt động chính xác 100% khi lên Producton.

## Hệ quả (Consequences)

### Tích cực:
1.  **Đồng nhất môi trường tuyệt đối:** Test local hoạt động chuẩn xác 100% như Production. Tránh các lỗi tiềm ẩn khi chuyển đổi giữa In-memory và Redis thật.
2.  **Tối ưu băng thông cực đại:** Chỉ những Pod thực sự giữ kết nối của người dùng mới nhận tin từ Redis. Cực kỳ tối ưu khi hệ thống mở rộng lên hàng triệu người dùng online đồng thời.
3.  **Quản lý tài nguyên an toàn:** Reference Counting bảo vệ Redis khỏi việc bị quá tải bởi các subscribe/unsubscribe request lặp đi lặp lại.

### Tiêu cực:
1.  **Yêu cầu hạ tầng local:** Lập trình viên bắt buộc phải có Docker Desktop hoặc WSL chạy Docker trên máy để khởi tạo container Redis trước khi chạy ứng dụng Spring Boot. Tuy nhiên, ranh giới này hoàn toàn chấp nhận được và là quy chuẩn trong phát triển ứng dụng Microservices hiện đại.
