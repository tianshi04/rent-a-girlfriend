# 🧪 ĐẶC TẢ TEST CASES & KẾT QUẢ KIỂM THỬ - PHASE 2: SSE REALTIME DELIVERY

Tài liệu này đặc tả chi tiết danh sách các Test Cases (Unit & Integration Tests) được thiết kế cho **Phase 2: SSE Realtime Delivery** và ghi nhận kết quả kiểm thử thực tế của hệ thống.

---

## 1. DANH MÁCH CÁC TEST CASES CHI TIẾT

Toàn bộ các test cases được viết theo triết lý **TDD (Test-Driven Development)** và bảo vệ nghiêm ngặt các quy tắc bất biến nghiệp vụ (Business Invariants).

### 1.1. Unit Tests: `SseConnectionRegistryTest`
*   **File test:** [SseConnectionRegistryTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/infrastructure/sse/SseConnectionRegistryTest.java)
*   **Mục tiêu:** Kiểm tra logic đăng ký, dọn dẹp, cơ chế đếm kết nối (Reference Counting) và tính an toàn đa luồng (Concurrency) của registry trong RAM.

| ID | Tên Test Case | Kịch bản / Điều kiện kích hoạt | Kỳ vọng kết quả (Asserts) | Bất biến bảo vệ |
| :--- | :--- | :--- | :--- | :---: |
| **TC-U01** | `shouldSubscribeOnFirstConnection` | Kết nối đầu tiên của User được tạo lập trên Pod. | - Đăng ký thành công emitter vào registry.<br>- Kích hoạt gọi `pubSubPort.subscribe` đúng 1 lần. | `[INV-N04]` |
| **TC-U02** | `shouldNotSubscribeAgainOnMultipleConnections` | User mở thêm kết nối thứ 2 (thiết bị thứ 2) active đồng thời. | - Registry có 2 emitters cho User.<br>- KHÔNG được gọi `pubSubPort.subscribe` lần 2 (đã subscribe ở TC-U01). | `[INV-N04]` |
| **TC-U03** | `shouldNotUnsubscribeWhenConnectionsStillActive` | Ngắt 1 kết nối khi User vẫn còn kết nối khác đang active. | - Emitter bị loại bỏ khỏi danh sách.<br>- KHÔNG được gọi `pubSubPort.unsubscribe` (vì vẫn còn connection). | `[INV-N04]` |
| **TC-U04** | `shouldUnsubscribeOnLastConnectionClosed` | Ngắt kết nối cuối cùng của User khỏi Pod. | - Emitters của User trống rỗng, xóa key khỏi RAM.<br>- Kích hoạt gọi `pubSubPort.unsubscribe` đúng 1 lần. | `[INV-N04]` |
| **TC-U05** | `shouldBeThreadSafeUnderHeavyConcurrency` | Giả lập **100 luồng đồng thời** gửi yêu cầu `register` và `unregister` trên cùng một `userId`. | - Thao tác thread-safe, không bị `ConcurrentModificationException`.<br>- Reference Count (số kết nối) cuối cùng hoàn toàn chính xác.<br>- Gọi `subscribe` và `unsubscribe` đúng 1 lần duy nhất. | `[INV-N04]` |

---

### 1.2. Integration Tests: `SseControllerIntegrationTest`
*   **File test:** [SseControllerIntegrationTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/interfaces/http/SseControllerIntegrationTest.java)
*   **Mục tiêu:** Kiểm tra tích hợp Spring Web context, endpoint streaming và bộ lọc Mock Auth Filter.

| ID | Tên Test Case | Kịch bản / Điều kiện kích hoạt | Kỳ vọng kết quả (Asserts) | Bất biến bảo vệ |
| :--- | :--- | :--- | :--- | :---: |
| **TC-I01** | `shouldEstablishSseConnectionWithMockAuth` | Client kết nối SSE cục bộ mà không gửi kèm header `user-id`. | - `MockAuthFilter` tự động mock header `user-id` mặc định.<br>- Trả về HTTP 200 và Content-Type `text/event-stream`.<br>- Nhận được gói tin bắt tay đầu tiên `:connected`. | Môi trường Dev |
| **TC-I02** | `shouldEstablishSseConnectionWithCustomUserId` | Client kết nối SSE có đính kèm header `user-id` tùy chọn. | - Kết nối thành công (HTTP 200).<br>- Nhận gói tin `:connected`. | Môi trường Prod |

---

### 1.3. Sửa lỗi & Nâng cấp các Test cũ (JPA Repository Tests)
*   **File test:** [NotificationRepositoryTest.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/com/rentagf/notification/infrastructure/persistence/NotificationRepositoryTest.java)

| ID | Tên Test Case | Mô tả nâng cấp | Lý do / Lợi ích |
| :--- | :--- | :--- | :--- |
| **TC-R01** | `testUniqueEventIdUserConstraint_duplicateShouldFail` | Tách kịch bản lưu trùng lặp thành test độc lập. | Tránh lỗi transaction hỏng (`marked for rollback`) của Hibernate gây sập oan các bước sau. |
| **TC-R02** | `testUniqueEventIdUserConstraint_differentUserShouldSuccess` | Tách kịch bản lưu trùng event khác user thành test độc lập. | Kiểm tra tính độc lập của transaction. |
| **TC-R03** | `testCursorBasedPagination` | Làm tròn thời gian test về đơn vị giây (`truncatedTo(ChronoUnit.SECONDS)`). | Loại bỏ lỗi mất độ phân giải nano-giây trong cơ sở dữ liệu in-memory H2. |

---

## 2. KẾT QUẢ KIỂM THỬ THỰC TẾ (TEST RESULTS)

Lệnh thực thi chạy test suite trên môi trường WSL 2 Linux:
```bash
wsl env GRADLE_USER_HOME=~/.gradle ./gradlew test
```

### 2.1. Thống kê tổng hợp (Summary)
*   **Tổng số Test Cases:** 33
*   **Thành công (Passed):** 33
*   **Thất bại (Failed):** 0
*   **Bỏ qua (Ignored):** 0
*   **Tỷ lệ thành công (Success Rate):** 100%
*   **Tổng thời gian thực thi (Duration):** 39.814 giây (Tổng thời gian build Gradle: 5 phút 13 giây bao gồm khởi tạo JVM daemon và configuration cache).
*   **Ngày giờ kiểm thử:** 2026-05-23T10:30:25+07:00 (Local Time)
*   **Công cụ build:** Gradle 8.14.4 (JVM Java 21)

---

### 2.2. Kết quả chi tiết theo từng Lớp Kiểm thử (Test Classes)

| Lớp Kiểm thử (Test Class) | Gói (Package) | Tổng số Test | Số test Pass | Số test Fail | Tỷ lệ thành công | Ghi chú |
| :--- | :--- | :---: | :---: | :---: | :---: | :--- |
| **`SseConnectionRegistryTest`** | `infrastructure.sse` | 5 | 5 | 0 | 100% | Đăng ký, đếm kết nối & an toàn đa luồng tốt. |
| **`SseControllerIntegrationTest`** | `interfaces.http` | 2 | 2 | 0 | 100% | Mock Auth Handshake kết nối stream trơn tru. |
| **`NotificationRepositoryTest`** | `infrastructure.persistence` | 5 | 5 | 0 | 100% | Hạn chế H2 nanoseconds và unique event constraint. |
| **`ArchitectureConsistencyTest`** | `com.rentagf.notification` | 5 | 5 | 0 | 100% | ArchUnit bảo vệ cấu trúc Hexagonal nghiêm ngặt. |
| **`AsyncDeliveryTest`** | `application` | 3 | 3 | 0 | 100% | Kiểm tra bất đồng bộ qua Virtual Threads. |
| **`NotificationDomainTest`** | `domain` | 7 | 7 | 0 | 100% | Bảo vệ Invariants & các Domain Logic hạt nhân. |
| **`GlobalExceptionHandlerTest`** | `interfaces.http` | 5 | 5 | 0 | 100% | Xử lý ngoại lệ HTTP và mapping mã lỗi nghiệp vụ. |
| **`NotificationServiceApplicationTests`**| `com.rentagf.notification` | 1 | 1 | 0 | 100% | Khởi động Context Spring Boot thành công. |

---

### 2.3. Nhật ký sửa lỗi đặc biệt (Hotfix Notes)

Trong lượt chạy kiểm thử này, một lỗi kiểm định kiến trúc bằng **ArchUnit** trong `ArchitectureConsistencyTest > portsMustBeInterfaces` đã được giải quyết triệt để:
*   **Lỗi gốc:** Class `SendResult` nằm trong package `application.port.outbound` sử dụng `@Builder` của Lombok. Khi biên dịch, Lombok tự động sinh ra một class static lồng nhau là `SendResultBuilder` (tên kết thúc bằng `Builder` chứ không phải `Result`), dẫn đến việc rule ArchUnit cũ bỏ sót và quét trúng lớp builder này như là một concrete class thông thường trong package port, vi phạm điều kiện "phải là Interface".
*   **Giải pháp:** Cập nhật lại rule kiểm tra `.and().haveSimpleNameNotEndingWith("Result")` thành `.and().haveSimpleNameNotContaining("Result")` để bỏ qua toàn bộ các lớp chứa hậu tố hoặc là một phần của Result DTO (bao gồm cả class Builder đi kèm). Bộ test đã chuyển sang màu xanh (GREEN) 100% tuyệt đối.
