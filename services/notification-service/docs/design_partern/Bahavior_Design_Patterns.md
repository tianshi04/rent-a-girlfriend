# 🎯 BEHAVIORAL DESIGN PATTERNS

## 1. Strategy Pattern (Mẫu Chiến lược)

*   **Dẫn chứng trong dự án**:
    *   **Strategy Interface**: [NotificationSender.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/NotificationSender.java)
    *   **Concrete Strategies**: [EmailPort.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/EmailPort.java), [FcmPort.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/FcmPort.java), [SsePort.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/SsePort.java)

### 📝 Tại sao sử dụng và hoạt động thế nào?
*   **Vấn đề**: Notification Service hỗ trợ gửi thông báo qua nhiều kênh kỹ thuật khác nhau (SSE, FCM Push, Email SMTP). Mỗi kênh có logic kết nối, thư viện ngoài và cấu trúc dữ liệu gửi tin hoàn toàn khác nhau. Nếu gộp tất cả logic này vào Use Case gửi tin, mã nguồn sẽ bị phình to với các câu lệnh điều kiện `if-else` / `switch-case` phức tạp, vi phạm nguyên tắc Open/Closed Principle (OCP).
*   **Giải pháp**: 
    1.  Định nghĩa interface chung `NotificationSender` với phương thức `SendResult send(Notification)` và phương thức nhận diện kênh `DeliveryChannel getChannel()`.
    2.  Mỗi cổng kênh cụ thể (`EmailPort`, `FcmPort`, `SsePort`) kế thừa từ interface này. Các adapter ở tầng hạ tầng sẽ tự do triển khai thuật toán gửi tin đặc thù của kênh mà không ảnh hưởng lẫn nhau.

### 📈 Ưu điểm đạt được
*   **Open/Closed Principle (OCP)**: Thêm kênh gửi tin mới (ví dụ: SMS, Slack, Telegram) chỉ cần tạo mới một Strategy Component kế thừa `NotificationSender` mà không cần chỉnh sửa code Use Case cốt lõi.

---

## 2. Registry Pattern (Mẫu Đăng ký Tập trung)

*   **Dẫn chứng trong dự án**:
    *   **Registry**: [NotificationSenderRegistry.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/registry/NotificationSenderRegistry.java)
    *   **Client Call**: [NotificationApplicationService.java:L67](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/service/NotificationApplicationService.java#L67)

### 📝 Tại sao sử dụng và hoạt động thế nào?
*   **Vấn đề**: Use Case cần định tuyến động kênh gửi tin dựa trên yêu cầu runtime (ví dụ: kênh truyền vào là `SSE` hay `FCM`). Việc inject cứng từng Strategy Bean vào Use Case sẽ gây độ kết dính cao (Tight Coupling).
*   **Giải pháp**:
    1.  `NotificationSenderRegistry` đóng vai trò là một Registry quản lý tập trung toàn bộ Strategy.
    2.  Sử dụng cơ chế Dependency Injection của Spring để tự động thu thập toàn bộ các Strategy Bean kế thừa `NotificationSender` qua constructor (`List<NotificationSender> senderList`).
    3.  Lưu trữ chúng vào một `Map<DeliveryChannel, NotificationSender>` tại thời điểm khởi động và cung cấp phương thức truy xuất động cực kỳ an toàn: `Optional<NotificationSender> getSender(DeliveryChannel channel)`.

### 📈 Ưu điểm đạt được
*   **Loose Coupling & Auto-Discovery**: Use Case chỉ giao tiếp với Registry để lấy Strategy mong muốn tại thời điểm chạy. Khi thêm Strategy mới, Spring tự động quét và đăng ký vào Registry mà không cần cấu hình thủ công.

---

## 3. Domain Exception Pattern (Mẫu Ngoại lệ Nghiệp vụ)

*   **Dẫn chứng trong dự án**:
    *   **Base Domain Exception**: [NotificationDomainException.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/errors/NotificationDomainException.java)
    *   **Concrete Domain Exception**: [DuplicateEventException.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/errors/DuplicateEventException.java), [InvalidCursorException.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/errors/InvalidCursorException.java)
    *   **Application Usage**: [NotificationApplicationService.java:L39](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/service/NotificationApplicationService.java#L39), [FetchInboxService.java:L29](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/service/FetchInboxService.java#L29)

### 📝 Tại sao sử dụng và hoạt động thế nào?
*   **Vấn đề**: Khi xảy ra vi phạm các ràng buộc nghiệp vụ (Invariants) tại tầng Domain hoặc Application (ví dụ: trùng lặp mã idempotency key khi gửi tin), việc ném ra các exception chung của Java như `IllegalArgumentException` hay `RuntimeException` sẽ gây khó khăn cho việc phân loại lỗi. Tầng Interface (REST API/gRPC) không thể nhận biết đây là lỗi do dữ liệu người dùng gửi lên bị sai (HTTP 400), trùng lặp dữ liệu (HTTP 409 Conflict), hay lỗi hệ thống (HTTP 500). Điều này dẫn đến việc trả về mã lỗi 500 INTERNAL_ERROR một cách bừa bãi, làm giảm chất lượng trải nghiệm API.
*   **Giải pháp**:
    1.  Định nghĩa một lớp cha đại diện cho toàn bộ lỗi nghiệp vụ của Service: `NotificationDomainException` kế thừa từ `RuntimeException`, lưu trữ thêm thông tin `errorCode` chuyên biệt.
    2.  Tạo ra các Exception con đại diện cụ thể cho từng lỗi nghiệp vụ riêng biệt, ví dụ `DuplicateEventException` đại diện cho việc trùng lặp sự kiện gửi tin.
    3.  Tầng Application Service thực hiện kiểm tra Invariants (Idempotency Guard), khi phát hiện vi phạm sẽ ném ngay lập tức `DuplicateEventException` (Fail-Fast).
    4.  Tầng Interface (GlobalExceptionHandler) bắt exception chuyên biệt này và chuyển đổi một cách chính xác sang HTTP 409 Conflict kèm JSON Error Payload chứa đúng `errorCode` của nghiệp vụ.

### 📈 Ưu điểm đạt được
*   **Tường minh nghiệp vụ (Ubiquitous Language)**: Exceptions phản ánh trực tiếp ngôn ngữ nghiệp vụ của dự án (`DUPLICATE_EVENT` thay vì lỗi kỹ thuật DB Constraint).
*   **Trải nghiệm API xuất sắc**: API Client nhận được mã lỗi HTTP chuẩn chỉ (409 Conflict) kèm thông tin rõ ràng thay vì lỗi 500 chung chung.
*   **Tách biệt tầng giao tiếp và tầng nghiệp vụ (Decoupling)**: Tầng nghiệp vụ chỉ cần ném lỗi nghiệp vụ, việc dịch lỗi sang JSON hay HTTP status code là của tầng HTTP Adapter.

---

## 4. Orchestrator Pattern (Mẫu Điều phối Luồng)

*   **Dẫn chứng trong dự án**:
    *   **Orchestrator**: [AsyncNotificationDeliveryService.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/service/AsyncNotificationDeliveryService.java)
    *   **Transaction Helper**: [NotificationDeliveryTransactionService.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/service/NotificationDeliveryTransactionService.java)

### 📝 Tại sao sử dụng và hoạt động thế nào?
*   **Vấn đề**: Khi xử lý gửi tin bất đồng bộ, chúng ta cần phối hợp nhịp nhàng giữa việc thao tác cơ sở dữ liệu (Database) và gọi I/O mạng ngoại vi (FCM/Email). Nếu gom tất cả vào một transaction `@Transactional` dài, hệ thống sẽ bị treo Database do giữ connection quá lâu (Hikari Exhaustion).
*   **Giải pháp**: Tách biệt luồng ra làm 3 bước và điều phối tập trung tại Orchestrator `AsyncNotificationDeliveryService`:
    1.  Orchestrator gọi Transaction Helper thực hiện giao dịch ngắn 1 (Đọc Notification, tạo Attempt, lưu DB).
    2.  Orchestrator thực thi cuộc gọi mạng vật lý (ngoài transaction, block I/O tự do).
    3.  Orchestrator gọi Transaction Helper thực hiện giao dịch ngắn 2 (Cập nhật kết quả).
    4.  Orchestrator đưa ra quyết định thực thi các side-effects ngoài transaction (retry trì hoãn, fallback SSE sang FCM).

### 📈 Ưu điểm đạt được
*   **Single Responsibility & High Performance**: Cô lập hoàn toàn transaction DB ra khỏi gọi mạng I/O dài, giải phóng 100% rủi ro sập connection pool HikariCP dưới tải cao.
}
