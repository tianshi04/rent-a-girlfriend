# TEST.md - Phase 3 Test Summary (Chỉ các Test liên quan đến Phase 3)

## 1. Danh sách các Test được Viết mới / Refactor trong Phase 3

Phase 3 tập trung vào việc chuẩn hóa **Hexagonal Architecture**, tách biệt Ports & Adapters, cấu hình lại **Manual ACK cho Kafka**, và tối ưu hóa **Async Delivery qua Transaction-After-Commit Hook** để tránh lỗi rò rỉ luồng (Race Condition).

Dưới đây là các test suite trực tiếp được tạo mới hoặc refactor mạnh mẽ trong Phase 3:

| Test Suite | Loại test | Trạng thái | Nội dung kiểm thử (What & How) |
| :--- | :--- | :--- | :--- |
| **AsyncDeliveryTest** | Integration Test | 100% Pass (3/3) | **What**: Kiểm tra luồng gửi thông báo bất đồng bộ sau khi Transaction đã Commit thành công (`@TransactionalEventListener`).<br>**How**: Sử dụng `ApplicationEventPublisher` để push `NotificationReadyEvent`. Dùng `Awaitility` để chờ Virtual Thread xử lý xong và kiểm tra trạng thái `Notification` trong H2 DB chuyển từ `PENDING` sang `COMPLETED` hoặc `FAILED` tương ứng. |
| **KafkaConsumerIntegrationTest** | Integration Test (E2E) | 100% Pass (1/1) | **What**: Kiểm tra Consumer nhận CloudEvent từ Kafka, parse, dịch sang tiếng Việt và lưu DB thành công với cấu hình **Manual ACK** (`spring.kafka.listener.ack-mode: manual`).<br>**How**: Sử dụng `@EmbeddedKafka` để giả lập Kafka Broker thật, publish raw JSON CloudEvent của Finance Service lên topic `finance-events`. Đợi Consumer xử lý và kiểm chứng bản ghi trong Database. |
| **NotificationServiceApplicationTests** | Smoke Test | 100% Pass (1/1) | **What**: Kiểm tra toàn bộ Spring Context của Notification Service khởi tạo thành công và không bị xung đột bean.<br>**How**: Khắc phục lỗi dependency bằng cách Mock các Adapter hạ tầng (`ConnectionStatePort`, `PubSubPort`, `JavaMailSender`) và cấu hình loại bỏ các auto-configuration không cần thiết. |
| **SseControllerIntegrationTest** | Controller Integration Test | 100% Pass (2/2) | **What**: Kiểm tra endpoint truyền thông stream SSE (`/v1/notifications/stream`) hoạt động đúng với cơ chế MockAuthFilter phân tách.<br>**How**: Sử dụng `MockMvc` giả lập client kết nối, kiểm tra xem client có nhận được gói tin bắt tay `:connected` (comment của SSE) hay không khi có và không có header `user-id`. |
| **NotificationRepositoryTest** | Repository Unit Test | 100% Pass (5/5) | **What**: Kiểm tra tích hợp lưu trữ JPA với H2 in-memory DB.<br>**How**: Được sửa lỗi Context loading bằng cách thêm cấu hình SMTP host ảo trong `tests/application.yml` để tránh lỗi thiếu cấu hình Mail của Actuator Health Check. |

---

## 2. Trả lời câu hỏi: Các Test thiếu này nên làm ở Phase nào?

Các phần test còn thiếu hiện tại bao gồm:
1. **Email sending (JavaMailSender)** ❌ (Hiện tại đang `@MockBean JavaMailSender`)
2. **FCM (Firebase Cloud Messaging) outbound** ❌ (Hiện tại đang `@MockBean FcmPort`)
3. **Push notification via custom provider** ❌ (Hiện tại đang `@MockBean SsePort` hoặc các provider khác)

### 💡 Đề xuất Phase thực hiện:

Các test này thuộc nhóm **Integration Test cho Outbound Adapters (Hạ tầng giao tiếp bên ngoài)**. Chúng không nên làm ở Phase 3 (kiến trúc cốt lõi), mà nên được triển khai ở **Phase 4: Integration & Gateway / Production Ready** (hoặc một Phase chuyên biệt về Outbound Adapters / Third-party Integrations).

#### Lý do cụ thể:
* **Email sending (JavaMailSender)**:
  * *Nên làm ở:* **Phase 4** hoặc **Phase Setup CI/CD**.
  * *Lý do:* Cần tích hợp với một Test SMTP server thực tế trong môi trường test (như **GreenMail** chạy trong Docker hoặc cấu hình in-memory). Việc này đòi hỏi setup thêm thư viện phụ thuộc, cấu hình test profile riêng để không gửi mail thật ra ngoài.
* **FCM (Firebase Cloud Messaging) outbound**:
  * *Nên làm ở:* **Phase 4 (Third-party Integration Test)**.
  * *Lý do:* FCM yêu cầu file config JSON (`firebase-adminsdk.json`) và cần kết nối tới Google API. Trong môi trường CI/CD hoặc local test, chúng ta phải viết Test bằng cách mock HTTP Client gọi tới endpoint Google FCM (sử dụng **WireMock** để mock REST API của Google FCM) thay vì gọi thật để tránh tốn quota hoặc cần credential thật.
* **Push notification via custom provider / SSE**:
  * *Nên làm ở:* **Phase 4**.
  * *Lý do:* Đảm bảo kiểm tra tính chịu tải của SSE Registry khi có hàng ngàn connection đồng thời (Performance/Load Test), cũng như kết nối cluster Redis PubSub thực tế khi chạy môi trường phân tán.

> **Kết luận**: Việc mock các outbound adapters này ở Phase 3 là hoàn toàn hợp lý và đúng chuẩn **TDD & Hexagonal Architecture** (ở Phase này ta chỉ cần chứng minh Application Core tương tác đúng với Port interface, còn việc Adapter thật nói chuyện với bên ngoài thế nào sẽ được kiểm thử ở Phase tích hợp chi tiết hơn).
