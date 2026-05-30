
# 🧪 NOTIFICATION SERVICE - TESTS CATALOG

Tài liệu này cung cấp mục lục chi tiết, kịch bản kiểm chứng và hướng dẫn chạy toàn bộ bộ kiểm thử (Test Suite) của **Notification Service** nhằm đảm bảo chất lượng mã nguồn hướng tới môi trường Production.

---

## 🎯 Chiến lược Kiểm thử (Testing Strategy)

Hệ thống áp dụng phương pháp **Test-Driven Development (TDD)** kết hợp kiểm thử phân lớp theo mô hình **Kiến trúc Lục giác (Hexagonal Architecture)**:
1. **Unit Tests (Kiểm thử đơn vị)**:
   - Tập trung vào tính độc lập, thuần Java của tầng Domain nghiệp vụ.
   - Mock các thành phần mạng/đối ngoại ở tầng Interfaces để kiểm chứng định dạng phản hồi lỗi.
2. **Integration Tests (Kiểm thử tích hợp)**:
   - Kiểm thử tích hợp thực tế với cơ sở dữ liệu PostgreSQL (lưu cascade, kiểm tra unique constraint).
   - Kiểm chứng khả năng xử lý bất đồng bộ đa luồng trên Java 21 Virtual Threads với thư viện `Awaitility`.

---

## 📚 Mục lục & Kịch bản Kiểm thử

### 1. [GlobalExceptionHandlerTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/interfaces/http/GlobalExceptionHandlerTest.java)
* **Loại test**: WebMvc Unit Test (MockMvc).
* **Mục tiêu**: Đảm bảo 100% các lỗi nghiệp vụ và lỗi hệ thống được ánh xạ sang định dạng lỗi lồng nhau JSON khớp hoàn toàn đặc tả API Contract (`api-contract.md §3`).
* **Các kịch bản kiểm chứng**:
  - `testHandleNotFoundException`: Trả về HTTP 404 với mã lỗi `NOTIFICATION_NOT_FOUND`.
  - `testHandleDuplicateNotificationException`: Trả về HTTP 409 với mã lỗi `DUPLICATE_EVENT` khi trùng lặp thông báo (idempotency key).
  - `testHandleAlreadyCompletedException`: Trả về HTTP 409 với mã lỗi `NOTIFICATION_ALREADY_COMPLETED`.
  - `testHandleRetryLimitExceededException`: Trả về HTTP 422 với mã lỗi `RETRY_LIMIT_EXCEEDED` khi quá 3 lần gửi lỗi.
  - `testHandleUnexpectedException`: Trả về HTTP 500 với mã lỗi chung `INTERNAL_ERROR` để tránh lộ thông tin DB nhạy cảm.

### 2. [NotificationDomainTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/domain/NotificationDomainTest.java)
* **Loại test**: Pure Domain Unit Test (Không Spring Context).
* **Mục tiêu**: Bảo vệ tuyệt đối các Invariants nghiệp vụ cốt lõi ở mức Domain Model thuần Java.
* **Các kịch bản kiểm chứng**:
  - `testInitialNotificationState`: Xác nhận trạng thái ban đầu của Notification mới tạo là `PENDING` và danh sách attempts rỗng.
  - `testCreateAttemptSuccessfully`: Tạo nỗ lực gửi tin mới, trạng thái chuyển sang `PROCESSING`.
  - `testMarkAttemptSuccess`: Đánh dấu attempt thành công, trạng thái Notification chuyển thành `COMPLETED` (Terminal State).
  - `testMarkAttemptFailedUnrecoverable`: Gửi tin thất bại với lỗi Unrecoverable -> chuyển trạng thái Notification sang `FAILED` ngay lập tức.
  - `testMarkAttemptFailedRecoverableKeepProcessing`: Gửi tin thất bại với lỗi Recoverable -> trạng thái giữ nguyên `PROCESSING` để sẵn sàng retry.
  - `testInvariantN01MaxRetriesExceeded` `[INV-N01]`: Thử gửi 3 lần thất bại (lỗi Recoverable) -> Notification tự động chuyển sang `FAILED`. Cố tạo thêm lần thứ 4 sẽ ném `RetryLimitExceededException`.
  - `testInvariantN02CannotAttemptAfterCompleted` `[INV-N02]`: Cố tạo thêm nỗ lực gửi mới sau khi Notification đã ở trạng thái `COMPLETED` sẽ ném `NotificationAlreadyCompletedException`.

### 3. [NotificationRepositoryTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/infrastructure/persistence/NotificationRepositoryTest.java)
* **Loại test**: JPA Database Integration Test.
* **Mục tiêu**: Xác thực tính toàn vẹn dữ liệu, quan hệ cascade và cơ chế phân trang.
* **Các kịch bản kiểm chứng**:
  - `testSaveAndFindNotificationSuccessfully`: Kiểm chứng lưu cascade, lưu Notification tự động lưu danh sách nỗ lực gửi `delivery_attempts` liên kết.
  - `testUniqueIdempotencyUserConstraint` `[INV-N03]`: Lưu 2 thông báo trùng `idempotency_key` và `user_id` sẽ văng lỗi `DataIntegrityViolationException` ở mức DB nhằm bảo vệ tính Idempotency. Đảm bảo cho phép lưu trùng `idempotency_key` cho 2 user khác nhau.
  - `testCursorBasedPagination`: Kiểm thử giải thuật phân trang Cursor-based Pagination theo `createdAt DESC, id DESC` sử dụng Spring Data JPA Limit API.
  - `testCountUnreadAndMarkAsRead`: Đếm số tin chưa đọc chính xác và kiểm chứng cơ chế cập nhật trạng thái đã đọc hàng loạt.

### 4. [AsyncDeliveryTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/application/AsyncDeliveryTest.java)
* **Loại test**: Multi-Threaded Async Integration Test.
* **Mục tiêu**: Đảm bảo luồng chạy bất đồng bộ `@Async` tận dụng trọn vẹn sức mạnh của **Java 21 Virtual Threads** không gây block luồng chính và ghi nhận đúng trạng thái cuối cùng xuống Database.
* **Các kịch bản kiểm chứng**:
  - `testSuccessfulAsyncDelivery`: Gửi tin thành công. Luồng chính trả về ngay lập tức, Awaitility chờ luồng Virtual Thread chạy nền hoàn tất việc cập nhật DB sang trạng thái `COMPLETED` thành công.
  - `testAsyncDeliveryUnrecoverableFailureDirectlyFails`: Gửi tin lỗi Unrecoverable trong luồng nền. Xác nhận DB chuyển trạng thái sang `FAILED` và cập nhật chính xác mã lỗi kỹ thuật từ Strategy Outbound Port.
  - `testDuplicateEventExceptionThrown`: Kiểm chứng tính Idempotency của API trigger. Khi gửi thông báo trùng lặp (trùng `idempotencyKey` và `userId`), hệ thống ném ra `DuplicateEventException`.

### 5. [ArchitectureConsistencyTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/ArchitectureConsistencyTest.java)
* **Loại test**: Architecture Unit Test (sử dụng thư viện **ArchUnit**).
* **Mục tiêu**: Đảm bảo tính bất biến của kiến trúc Ports & Adapters, tự động kiểm soát import và ranh giới hệ thống để ngăn chặn 100% các lỗi rò rỉ framework (Spring Web, SseEmitter, JPA) vào Core Application.
* **Các kịch bản kiểm chứng (Quy tắc kiến trúc)**:
  - `coreShouldBeFreeOfSpringWebFramework`: Quét Core (Application + Domain) để chặn đứng mọi import của Spring Web (`org.springframework.web..` và `SseEmitter`).
  - `coreShouldBeFreeOfJpaDetail`: Chặn đứng mọi rò rỉ của công nghệ DB (`jakarta.persistence..`) vào lõi nghiệp vụ.
  - `domainShouldBeCompletelyIndependent`: Đảm bảo lớp Domain hoàn toàn độc lập, không được phép import bất cứ class nào từ các package bên ngoài.
  - `applicationShouldNotDependOnInfrastructureOrInterfaces`: Cấm tuyệt đối việc Application Layer phụ thuộc vào Infrastructure hay Interfaces.
  - `portsMustBeInterfaces`: Ràng buộc tất cả các cổng Port khai báo ở package `application.port` phải là **Interface** theo đúng nguyên lý Dependency Inversion.

---

## 🚀 Hướng dẫn Chạy Kiểm thử

Do máy host của nhà phát triển có thể chưa được cấu hình Java 21 JDK hoặc database PostgreSQL cục bộ, toàn bộ bộ kiểm thử được thiết kế để chạy hoàn toàn cô lập, an toàn và sạch sẽ bên trong container Docker.

### 1. Chạy Toàn bộ Bộ kiểm thử (Full Test Suite)
Mở terminal (PowerShell trên Windows) tại thư mục `services/notification-service` và thực hiện lệnh:
```powershell
docker run --rm -v "${PWD}:/app" -w /app eclipse-temurin:21-jdk-alpine ./gradlew test
```

### 2. Chạy một File kiểm thử cụ thể (Ví dụ: Domain Test)
```powershell
docker run --rm -v "${PWD}:/app" -w /app eclipse-temurin:21-jdk-alpine ./gradlew test --tests com.rentagf.notification.domain.NotificationDomainTest
```

### 3. Chạy kiểm thử Repository hoặc Async Test
```powershell
docker run --rm -v "${PWD}:/app" -w /app eclipse-temurin:21-jdk-alpine ./gradlew test --tests com.rentagf.notification.infrastructure.persistence.NotificationRepositoryTest
```
*(Lưu ý: Chạy các bài test Integration yêu cầu cấu hình Database test đang hoạt động hoặc được cấu hình tự động thông qua Spring Boot).*
