# ADR 0010: Refactoring SSE Connection Management - Strict Hexagonal Architecture & Framework Isolation

## Trạng thái (Status)
Đã duyệt (Accepted)

## Bối cảnh (Context)

### 1. Hiện trạng thiết kế ban đầu và lý do "Đi tắt" (Pragmatic Shortcut)
Trong Phase 1 và giai đoạn đầu Phase 2, để nhanh chóng xây dựng luồng truyền tin thời gian thực Server-Sent Events (SSE), đội ngũ phát triển đã lựa chọn giải pháp đi tắt (short-circuiting):
*   **Browser/Client** gọi HTTP GET đến `SseController` (Inbound Adapter).
*   `SseController` khởi tạo đối tượng `SseEmitter` (Spring Web Framework) và gọi trực tiếp đến `SseConnectionRegistry` để đăng ký kết nối cục bộ.
*   `SseConnectionRegistry` được đặt trong package `com.rentagf.notification.application.registry`.

**Lập luận của việc đi tắt ban đầu:**
*   Nghĩ rằng việc thiết lập stream kết nối SSE chỉ đơn thuần là tác vụ kỹ thuật I/O mạng (network transport), không chứa logic nghiệp vụ (business logic) của ứng dụng nên không cần thiết phải đi qua một UseCase/Inbound Port của Application Layer làm tăng thêm boilerplate code (code mẫu thừa).
*   Coi `SseConnectionRegistry` như một component lưu giữ trạng thái session chung, nên đặt nó ở lớp `application` để cả Inbound Adapter (`SseController`) và Outbound Adapter (`RedisPubSubAdapter`) đều có thể gọi trực tiếp.

### 2. Vấn đề rò rỉ kiến trúc nghiêm trọng (Architectural Leak)
Thiết kế "đi tắt" trên đã dẫn đến một điểm "Smell" kiến trúc rất nặng:
1.  **Nhiễm độc Framework (Framework Coupling):** `SseConnectionRegistry` (đang nằm ở tầng `application`) phải import và sử dụng trực tiếp class `SseEmitter` của Spring Boot Web (`org.springframework.web.servlet.mvc.method.annotation.SseEmitter`). Điều này vi phạm nghiêm trọng nguyên lý Hexagonal/Clean Architecture: **Tầng lõi Application & Domain phải hoàn toàn thuần khiết (Pure Java) và độc lập tuyệt đối với Web Framework.**
2.  **Khó Unit Test:** Việc tầng `application` bị phụ thuộc chặt vào Spring Web khiến ta không thể viết Unit Test độc lập cho các Use Case nghiệp vụ mà không phải khởi dựng hoặc mock các thành phần của Web Servlet.
3.  **Vi phạm ranh giới Inbound:** `SseController` (Inbound Adapter) giao tiếp trực tiếp với một component cụ thể của hệ thống mà không đi qua bất kỳ **Inbound Port** nào làm ranh giới bảo vệ.

---

## Quyết định (Decision)

Để đảm bảo Notification Service đạt tiêu chuẩn kiến trúc sẵn sàng cho môi trường Production, dễ bảo trì và kiểm thử, chúng ta quyết định thực hiện **Refactor toàn diện luồng kết nối SSE theo nguyên lý Strict Hexagonal Architecture** với các điểm mấu chốt sau:

### 1. Cô lập hoàn toàn `SseEmitter` ra khỏi Application Core
*   Di chuyển toàn bộ class `SseConnectionRegistry` về đúng vị trí thực tế của nó dưới tầng hạ tầng:
    *   *Package mới:* `com.rentagf.notification.infrastructure.sse.SseConnectionRegistry` (hoặc `infrastructure.adapter.sse`).
*   **Nguyên tắc:** Lớp lõi `application` và `domain` tuyệt đối không được phép chứa bất kỳ import nào liên quan đến `SseEmitter` hoặc `org.springframework.web`.

### 2. Thiết lập Inbound Port (UseCase) cho luồng đăng ký kết nối
*   Định nghĩa một Inbound Port mới có tên **`NotificationSubscriptionUseCase`**:
    ```java
    package com.rentagf.notification.application.port.inbound;
    import java.util.UUID;
    
    public interface NotificationSubscriptionUseCase {
        void subscribe(UUID userId);
    }
    ```
*   `SseController` (Inbound Adapter) sau khi bắt tay kết nối và khởi tạo `SseEmitter`, sẽ chỉ gọi vào Inbound Port này:
    ```java
    // Trong SseController
    subscriptionUseCase.subscribe(userId);
    ```
*   Tầng Application Service (`NotificationApplicationService`) sẽ triển khai (implement) Inbound Port này để xử lý các logic nghiệp vụ khi có user kết nối (ví dụ: truy vấn danh sách thông báo chưa đọc từ DB để chuẩn bị push, cập nhật trạng thái online của user, ghi log audit).

### 3. Quy trình gửi tin nhắn khép kín thông qua Outbound Port & Adapter
Khi Application Service cần đẩy tin nhắn thời gian thực:
1.  Nó gọi qua Outbound Port `SsePort` (định nghĩa giao ước gửi tin bằng các đối tượng Domain thuần túy).
2.  `SseOutboundAdapter` (Outbound Adapter - Infrastructure) thực thi `SsePort`, đóng gói tin nhắn và publish lên Redis Pub/Sub.
3.  `RedisPubSubAdapter` (lắng nghe sự kiện từ Redis) sẽ truy cập trực tiếp vào `SseConnectionRegistry` (bây giờ đã nằm hoàn toàn ở tầng Infrastructure) để lấy đối tượng `SseEmitter` vật lý và thực hiện lệnh `emitter.send()`.

---

## Hệ quả (Consequences)

### Tích cực (Pros):
1.  **Kiến trúc sạch tuyệt đối (Zero Framework Leak):** Tầng Application Core hoàn toàn thuần khiết, không chứa bất kỳ import nào của Spring Web Framework. Cực kỳ dễ dàng migrate framework hoặc nâng cấp phiên bản mà không sợ ảnh hưởng đến core nghiệp vụ.
2.  **Khả năng kiểm thử (Testability) tối đa:** Dễ dàng viết Unit Test độc lập cho `NotificationApplicationService` bằng cách Mock Inbound/Outbound Ports mà không cần dính dáng gì đến Spring Web context hay `SseEmitter`.
3.  **Tách biệt trách nhiệm rõ ràng:** 
    *   Tầng Application chỉ lo logic nghiệp vụ (ai đăng ký, gửi tin gì).
    *   Tầng Infrastructure lo chi tiết kỹ thuật mạng (lưu trữ emitter ở đâu, quản lý luồng, timeout, kết nối Redis như thế nào).

### Tiêu cực (Cons):
1.  **Tăng nhẹ lượng file trung gian (Boilerplate):** Cần tạo thêm interface `NotificationSubscriptionUseCase` và cấu hình injection tương ứng. Tuy nhiên, cái giá này hoàn toàn xứng đáng để đổi lấy sự an toàn và bền vững của kiến trúc hệ thống lớn.
