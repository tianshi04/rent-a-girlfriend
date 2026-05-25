# ADR 0007: Chiến lược Gửi tin Ngoại vi, Phân loại Lỗi và Xử lý Bất đồng bộ (Outbound Delivery, Error Handling, and Async Strategy)

**Trạng thái:** Chấp nhận (Accepted)  
**Ngày:** 2026-05-21  

---

## 1. Ngữ cảnh (Context)

Trong thiết kế Skeleton ban đầu (Phase 0), cơ chế gửi tin ngoại vi đang gặp 3 vấn đề lớn về mặt kiến trúc đối với một hệ thống **Production-grade**:

1. **Thiếu thông tin kết quả gửi**: Các interface Port (`FcmPort`, `EmailPort`, `SsePort`) chỉ trả về kiểu `boolean` (thành công hoặc thất bại). Kiểu dữ liệu này "quá nghèo thông tin", khiến Domain Aggregate Root (`Notification`) không có đủ dữ liệu để đưa ra các quyết định nghiệp vụ quan trọng (ví dụ: Lỗi này có thể thử lại được không? Mã lỗi của Firebase/SMTP là gì để lưu audit trace? Message ID từ nhà cung cấp là gì để đối soát?).
2. **Retry Policy bị đánh đồng**: Mỗi kênh gửi có đặc thù truyền thông và chính sách thử lại (Retry) hoàn toàn khác nhau. SSE chạy trên socket mở không thể retry theo kiểu backoff (nếu đứt kết nối là user đã offline, phải fallback sang FCM ngay). FCM và Email có thể retry nhưng chỉ được phép retry đối với các lỗi tạm thời (Recoverable như Timeout, 503) và cấm retry đối với lỗi logic (Unrecoverable như Token hết hạn, Email sai định dạng).
3. **Luồng xử lý đồng bộ gây nghẽn (Sync Blocking)**: Nếu UseCase thực hiện gửi tin đồng bộ (Sync) trực tiếp tới các API bên ngoài (Firebase, SMTP), thời gian phản hồi (Response time) của UseCase sẽ bị kéo dài (thậm chí bị timeout nếu API bên thứ 3 phản hồi chậm). Điều này làm sụt giảm nghiêm trọng thông năng (Throughput) xử lý sự kiện của hệ thống.

---

## 2. Quyết định Kiến trúc (Decisions)

Để giải quyết triệt để các vấn đề trên, hệ thống quyết định áp dụng các giải pháp kiến trúc sau:

### Quyết định 2.1: Chuẩn hóa kết quả gửi tin bằng lớp `SendResult`

Thay thế hoàn toàn kiểu trả về `boolean` bằng lớp đối tượng giàu ngữ cảnh `SendResult` ở các Outbound Ports:

```java
package com.rentagf.notification.application.port.outbound;

import lombok.Builder;
import lombok.Getter;

@Getter
@Builder
public class SendResult {
    private final boolean success;
    private final String messageId;        // ID tin nhắn từ nhà cung cấp (FCM, SendGrid...) để đối soát (Audit)
    private final String errorCode;        // Mã lỗi hệ thống (ví dụ: FCM_TOKEN_INVALID, SMTP_TIMEOUT)
    private final String errorMessage;     // Chi tiết thông báo lỗi phục vụ debug
    private final boolean recoverable;     // Lỗi tạm thời có thể cứu vãn (true) hay lỗi vĩnh viễn (false)

    public static SendResult success(String messageId) {
        return SendResult.builder()
                .success(true)
                .messageId(messageId)
                .recoverable(false)
                .build();
    }

    public static SendResult fail(String errorCode, String errorMessage, boolean recoverable) {
        return SendResult.builder()
                .success(false)
                .errorCode(errorCode)
                .errorMessage(errorMessage)
                .recoverable(recoverable)
                .build();
    }
}
```

---

### Quyết định 2.2: Phân tách và cấu hình Retry Policy theo Kênh gửi

Định nghĩa chính sách Retry rõ ràng cho từng kênh gửi:

1. **Kênh SSE (Realtime Socket)**:
   - **Retry Policy**: **Không áp dụng Retry**. 
   - **Hành vi**: Nếu việc đẩy tin qua SSE thất bại (văng lỗi `Broken pipe`, client offline), hệ thống lập tức thực hiện **FCM Fallback** hoặc chỉ lưu vào Inbox DB.
2. **Kênh FCM (Firebase Push)**:
   - **Retry Policy**: Thực hiện tối đa **3 lần** thử lại.
   - **Thuật toán**: **Exponential Backoff** (chờ 2s -> 4s -> 8s) để tránh làm quá tải hệ thống Firebase.
   - **Điều kiện**: Chỉ thực hiện retry khi nhận được `SendResult` có `recoverable = true` (ví dụ: `503 Service Unavailable`, `Timeout`). Gặp lỗi `recoverable = false` (như token hết hạn, sai cấu hình payload) -> Đánh dấu FAILED ngay lập tức.
3. **Kênh Email (SMTP)**:
   - **Retry Policy**: Thực hiện tối đa **3 lần** thử lại.
   - **Thuật toán**: **Exponential Backoff** (chờ 5s -> 15s -> 45s).
   - **Điều kiện**: Chỉ retry với lỗi `recoverable = true` (lỗi kết nối SMTP server).

---

### Quyết định 2.3: Thiết lập hàng đợi bất đồng bộ (Async Queue & Worker Pool)

Để cô lập hoàn toàn UseCase nghiệp vụ cốt lõi khỏi độ trễ mạng của các API ngoại vi, luồng dữ liệu sẽ được chuyển đổi sang bất đồng bộ:

```
[UseCase] 
  │
  ├── 1. Lưu Notification (status: PENDING) & DeliveryAttempt (status: PENDING)
  │
  ├── 2. Đẩy Job (Notification ID) vào Async Queue 
  │
  └── 3. Trả kết quả thành công ngay lập tức cho Event Subscriber (Không block!)
  
     ▲
     │ (Bất đồng bộ)
     ▼
[Worker Pool (Virtual Threads / ThreadPool)]
  │
  ├── 1. Nhặt Job từ Async Queue
  ├── 2. Gọi NotificationSender (Strategy) để gửi tin
  ├── 3. Nhận SendResult -> Cập nhật kết quả vào DB
  └── 4. Nếu thất bại & Recoverable -> Lên lịch Retry (Delayed Task)
```

- **Công nghệ áp dụng**: Sử dụng **Java 21 Virtual Threads** (kích hoạt qua cấu hình `spring.threads.virtual.enabled: true` trong Spring Boot 3.5.0) để chạy luồng gửi tin bất đồng bộ vô cùng nhẹ nhàng, tự động co giãn theo tải và giải phóng hoàn toàn việc quản lý Thread Pool thủ công.
- **Delayed Task cho Retry**: Sử dụng `ScheduledExecutorService` hoặc cơ chế Delayed Message của Kafka để lên lịch chạy lại sau khoảng thời gian backoff mà không block luồng Worker.

---

## 3. Hệ quả (Consequences)

### Tích cực (Positives)
- **Tốc độ phản hồi cực nhanh**: UseCase không bị block bởi các API gửi tin chậm chạp bên ngoài. Thông năng xử lý tin nhắn của hệ thống tăng gấp hàng chục lần.
- **Truy vết lỗi hoàn hảo**: Có đầy đủ thông tin `errorCode`, `errorMessage` và `messageId` trong bảng `delivery_attempts`, giúp đội vận hành dễ dàng giám sát lỗi hệ thống.
- **Bảo vệ tài nguyên**: Tiết kiệm tài nguyên mạng, ngăn ngừa việc retry vô nghĩa vào các token FCM đã chết.

### Đánh đổi / Tiêu cực (Trade-offs)
- **Kiểm thử phức tạp hơn**: Việc viết integration tests đòi hỏi phải sử dụng các thư viện kiểm thử bất đồng bộ (như `Awaitility`) để chờ đợi kết quả ghi nhận xuống DB ở luồng nền.
- **Trạng thái nhất quán muộn (Eventual Consistency)**: Khi API trả về thành công cho Client, thông báo có thể vẫn đang nằm trong hàng đợi chờ được gửi đi thực tế.
