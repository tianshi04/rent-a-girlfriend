# 🧪 KẾ HOẠCH KIỂM THỬ NÂNG CẤP (TESTS.md) - PHASE 5: OUTBOUND PROVIDERS & RETRY EXECUTION

Tài liệu này đóng vai trò là **Mục lục Kiểm thử (Test Directory)** và **Kịch bản Kiểm thử chi tiết (Test Specification)** cho các chức năng thuộc Phase 5 theo phương pháp **Test-Driven Development (TDD)**. 

---

## 🗺️ Mục lục các Lớp Kiểm thử

Các file kiểm thử sẽ được cấu trúc và tổ chức tương ứng với các layer trong kiến trúc Hexagonal:

1. **Unit Tests (Domain & Application Layers)**:
   - **`InMemoryRetrySchedulerTest.java` [NEW]**: Kiểm thử bộ lập lịch retry trì hoãn sử dụng in-memory executor.
   - **`AsyncNotificationDeliveryServiceTest.java` [MODIFY]**: Kiểm thử luồng gửi tin bất đồng bộ, chính sách Exponential Backoff, cơ chế xử lý lỗi (Recoverable vs Unrecoverable), và cơ chế Fallback từ SSE sang FCM.
2. **Integration Tests (Infrastructure & Persistence Layers)**:
   - **`FcmOutboundAdapterTest.java` [NEW]**: Kiểm thử Simulation Mode (không credentials) và Real Mode (kết nối Firebase SDK thật).
   - **`EmailOutboundAdapterTest.java` [NEW]**: Tích hợp với **GreenMail** (SMTP Server in-memory) để kiểm thử việc gửi email thật qua SMTP.
   - **`NotificationRetryIntegrationTest.java` [NEW]**: Kiểm thử tích hợp End-to-End quy trình Retry trì hoãn và Fallback thực tế từ Database lên tới Application Service.

---

## 🎯 Chi tiết Kịch bản Kiểm thử (Test Cases Specification)

### 1. Application Unit Tests

#### 1.1. `InMemoryRetrySchedulerTest.java`
Kiểm thử tính chính xác của bộ lên lịch retry trì hoãn in-memory.
- **`testScheduleRetry_DispatchesEventAfterDelay`**:
  - **Mục tiêu**: Đảm bảo sự kiện `NotificationReadyEvent` được phát đi đúng thời gian trễ đã định cấu hình.
  - **Mô phỏng**: Lên lịch retry sau 50ms.
  - **Kỳ vọng**: 
    - Lập tức kiểm tra: Chưa có sự kiện nào được phát đi.
    - Chờ 100ms: Kiểm tra xem sự kiện `NotificationReadyEvent` đã được phát thành công với đúng ID và đúng Channel chưa.
- **`testScheduleMultipleRetries_IndependentExecution`**:
  - **Mục tiêu**: Đảm bảo nhiều task retry được lên lịch song song hoạt động độc lập và không block nhau.

#### 1.2. `AsyncNotificationDeliveryServiceTest.java`
Kiểm tra luồng xử lý gửi tin bất đồng bộ, xử lý lỗi và retry.
- **`testSendAsync_Success_UpdatesStatusToCompleted`**:
  - **Mục tiêu**: Gửi thành công thông báo qua FCM/Email.
  - **Kỳ vọng**:
    - Tạo attempt mới ở trạng thái `SUCCESS`.
    - Trạng thái của Notification chuyển thành `COMPLETED`.
    - `resolvedAt` và `messageId` được lưu chính xác.
- **`testSendAsync_RecoverableFailure_SchedulesRetry`**:
  - **Mục tiêu**: Khi gửi tin bị lỗi recoverable (lần 1), hệ thống phải tự động lên lịch retry.
  - **Mock**: Adapter trả về `SendResult.fail("TIMEOUT", "SMTP Timeout", true)`.
  - **Kỳ vọng**:
    - Tạo attempt mới ở trạng thái `FAILED_RECOVERABLE`.
    - Trạng thái Notification giữ nguyên `PROCESSING`.
    - Gọi `RetrySchedulerPort.scheduleRetry` với khoảng thời gian delay tăng dần chính xác (ví dụ lần đầu FCM là 2 giây).
- **`testSendAsync_UnrecoverableFailure_NoRetryAndFailsNotification`**:
  - **Mục tiêu**: Khi gửi tin bị lỗi unrecoverable, hệ thống lập tức đóng thông báo là FAILED.
  - **Mock**: Adapter trả về `SendResult.fail("INVALID_EMAIL", "Email invalid", false)`.
  - **Kỳ vọng**:
    - Tạo attempt mới ở trạng thái `FAILED_UNRECOVERABLE`.
    - Trạng thái Notification chuyển thành `FAILED`.
    - **Không** gọi `RetrySchedulerPort.scheduleRetry`.
- **`testSendAsync_RetryLimitExceeded_FailsNotification`**:
  - **Mục tiêu**: Khi đạt giới hạn 3 lần retry thất bại (tổng cộng 4 lần gửi), thông báo phải chuyển sang trạng thái FAILED.
  - **Mô phỏng**: Notification đã có sẵn 3 attempts thất bại trong database.
  - **Kỳ vọng**:
    - Ném ra `RetryLimitExceededException` (Do `Notification.createAttempt` tự bảo vệ Invariant `[INV-N01]`).
    - Ghi nhận nỗ lực gửi thất bại và đóng trạng thái Notification thành `FAILED`.
- **`testSendAsync_SseFailure_TriggersFcmFallback`**:
  - **Mục tiêu**: Gửi SSE thất bại -> tự động chuyển kênh fallback sang FCM Push.
  - **Mock**: SSE Adapter trả về `SendResult.fail("SSE_CONNECTION_LOST", "Client disconnected", true)`.
  - **Kỳ vọng**:
    - Ghi nhận attempt SSE là `FAILED_UNRECOVERABLE` (vì SSE hỏng thì phiên SSE đó hỏng luôn).
    - Phát ra sự kiện `NotificationReadyEvent` cho kênh `FCM` bất đồng bộ để gửi push thay thế.

---

## 2. Infrastructure Integration Tests

#### 2.1. `FcmOutboundAdapterTest.java`
Kiểm thử khả năng tương tác với Firebase SDK.
- **`testSend_SimulationMode_Success`**:
  - **Điều kiện**: Chạy test trong môi trường thiếu file credentials Service Account JSON.
  - **Kỳ vọng**: 
    - Adapter tự động phát hiện và kích hoạt Simulation Mode.
    - Không crash ứng dụng. Trả về `SendResult` thành công với message ID dạng `"fcm-sim-*"`.
- **`testSend_RealMode_ErrorClassification`**:
  - **Điều kiện**: Khởi tạo Firebase SDK giả lập với credentials test.
  - **Kỳ vọng**: Phân loại chính xác các mã lỗi Firebase thành `recoverable = true` hoặc `false`.

#### 2.2. `EmailOutboundAdapterTest.java`
Kiểm thử tích hợp gửi mail SMTP thực tế sử dụng GreenMail.
- **Cấu hình Test**: Khởi tạo GreenMail SMTP server in-memory trên port ngẫu nhiên.
- **`testSendEmail_SmtpSuccess`**:
  - **Mục tiêu**: Gửi email thành công qua GreenMail SMTP.
  - **Kỳ vọng**:
    - `EmailOutboundAdapter` gửi thư không lỗi.
    - GreenMail nhận được email chuẩn xác (Khớp: From, To, Subject, Body).
- **`testSendEmail_SmtpTimeout_Recoverable`**:
  - **Mục tiêu**: Kiểm tra lỗi mạng SMTP được phân loại là recoverable.
  - **Mô phỏng**: Tắt GreenMail Server trước khi gửi.
  - **Kỳ vọng**:
    - Adapter bắt được `MailSendException`.
    - Trả về `SendResult` có `success = false` và `recoverable = true`.

#### 2.3. `NotificationRetryIntegrationTest.java`
Kiểm thử tích hợp End-to-End toàn bộ luồng Retry & Fallback.
- **`testEndToEnd_RetryUntilSuccess`**:
  - **Luồng kiểm thử**:
    1. Gửi thông báo Email -> SMTP offline -> Ghi nhận `FAILED_RECOVERABLE` -> Lên lịch retry.
    2. Bật SMTP Server lên.
    3. Chờ trigger retry -> Gửi lại thành công -> Trạng thái Notification cập nhật thành `COMPLETED` trong Database.
