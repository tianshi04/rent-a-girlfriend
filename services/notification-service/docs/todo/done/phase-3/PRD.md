# 📄 PRODUCT REQUIREMENT DOCUMENT (PRD) - PHASE 3: EVENT CONSUMER + ROUTING ENGINE

## 1. Tổng quan Dự án (Project Overview)
Phase 3 là bộ não của **Notification Service**. Nó kết nối các adapter realtime (SSE - Phase 2) và domain logic (Phase 1) với hệ thống thông điệp bất đồng bộ (Kafka/Event Broker). Mục tiêu chính là nhận các sự kiện nghiệp vụ từ toàn hệ thống, tự động dịch nghĩa thành nội dung trực quan, chọn kênh gửi tối ưu và phân phối tới người dùng cuối.

---

## 2. Các Tính năng & Yêu cầu Sản phẩm (Product Features)

### 2.1. Bộ lắng nghe Sự kiện & Giải mã (Event Consumer & CloudEvents Parser)
*   **Mô tả**: Lắng nghe các Kafka Topics để nhận các sự kiện nghiệp vụ dưới dạng CloudEvent v1.0.
*   **Yêu cầu kỹ thuật**:
    *   Subscribe các topics của các context: `booking-events`, `finance-events`, `interaction-events`, `profile-events`, `identity-events`, `dispute-events`, `notification-requested-events` (Hoặc subscribe wildcard tùy thuộc vào hạ tầng broker).
    *   Hỗ trợ CloudEvents Envelope chứa các trường: `specversion`, `id`, `source`, `type`, `time`, `datacontenttype`, `data`.
    *   Giải mã (Deserialize) `data` dạng JSON thành Map để dễ dàng xử lý.

### 2.2. Bộ lọc Trùng lặp (Idempotency Guard)
*   **Mô tả**: Bảo vệ người dùng khỏi việc nhận một thông báo nhiều lần do hạ tầng mạng phát lại (At-least-once delivery).
*   **Yêu cầu kỹ thuật**:
    *   Sử dụng CloudEvent `id` làm `eventId` (idempotency key).
    *   Kiểm tra sự tồn tại của cặp composite `(eventId, userId)` trong Database trước khi xử lý.
    *   Nếu phát hiện trùng lặp: Log warning dạng "Duplicate event detected for eventId {} and userId {}", từ chối xử lý tiếp và ACK message thành công cho Kafka (tránh block partition).

### 2.3. Lớp Dịch Sự kiện & Quản lý Template (Event Translator & Template Engine)
*   **Mô tả**: Chuyển đổi dữ liệu thô từ Domain Event thành nội dung thông báo có nghĩa (Tiêu đề, Nội dung) dựa trên cấu hình template tập trung.
*   **Yêu cầu kỹ thuật**:
    *   Tự động tải cấu hình `templates.yaml` tại thời điểm startup.
    *   **Ráp nội dung (Interpolate)**: Thực hiện thay thế các tham số `{{fieldName}}` bằng giá trị thực tế lấy từ Event `data` (Ví dụ: `{{bookingId}}` được thay bằng mã booking).
    *   **Ngôn ngữ hiển thị**: **Chỉ hỗ trợ tiếng Việt (`vi`)** từ template của `templates.yaml` trong Phase này. Bỏ hoàn toàn logic đa ngôn ngữ phức tạp để tối giản thiết kế.
    *   Hỗ trợ trích xuất **Recipient ID** động dựa trên cấu hình `recipient_field` của từng sự kiện trong `templates.yaml`.

### 2.4. Công cụ Định tuyến Thông minh (Routing Engine)
*   **Mô tả**: Quyết định kênh phân phối tối ưu (SSE, FCM Push, Email) dựa trên chính sách mặc định, trạng thái online của user, và cấu hình đè (policy overrides).
*   **Yêu cầu kỹ thuật**:
    *   **Quy tắc 1 (Email)**: Nếu `requireEmail == true` hoặc cấu hình template yêu cầu, gửi email ngay lập tức (Bất kể user Online/Offline) qua Mailtrap SMTP.
    *   **Quy tắc 2 (SSE First)**: Kiểm tra xem user có đang Online trên SSE hay không. Nếu Online -> Gửi qua SSE. Nếu source service có truyền cờ `requirePush == true`, gửi song song qua FCM Push.
    *   **Quy tắc 3 (FCM Fallback)**: Nếu user Offline -> Gửi FCM Push nếu `requirePush == true` hoặc độ ưu tiên thông báo là `HIGH` (TRANSACTIONAL HIGH). Nếu không, chỉ lưu vào Inbox Database ở trạng thái `PENDING` (User sẽ thấy khi mở app).
    *   **Quản lý trạng thái Online (Distributed Connections)**: Phải hỗ trợ môi trường phân tán đa Pods thông qua bộ lưu trữ trạng thái kết nối tập trung (Redis Connection State Store).
    *   **Tích hợp Outbound Service Thực tế**:
        *   **Email**: Tích hợp **Spring Boot Starter Mail** (JavaMailSender) thực tế. Cho phép cấu hình SMTP Server linh hoạt qua file `application.yml` để dễ dàng thử nghiệm gửi mail thật tới hộp thư ảo Mailtrap Sandbox.
        *   **FCM Push**: Tích hợp **Firebase Admin SDK** chính thức. Đọc file credentials JSON từ root dự án để kết nối và đẩy push notification thật tới các client. Nếu không có file credential, adapter sẽ chạy ở chế độ mô phỏng mạng để phục vụ kiểm thử cục bộ.

### 2.5. Kênh Bị động (Passive Channel Support)
*   **Mô tả**: Cho phép các dịch vụ khác gửi thông báo có nội dung đã được định nghĩa và format sẵn.
*   **Yêu cầu kỹ thuật**:
    *   Lắng nghe sự kiện `com.rentagf.notification.requested.v1`.
    *   Bỏ qua bước Translator, lấy trực tiếp `title`, `body`, `action_url`, `image_url` từ payload và đưa vào Routing Engine để deliver.

---

## 3. Các Ràng buộc & Tiêu chí Non-Functional (NFRs)
*   **Hiệu năng**: Xử lý một sự kiện từ khi nhận tới khi gửi realtime thành công qua SSE < 150ms.
*   **Tính sẵn sàng (Availability)**: Tách biệt hoàn toàn luồng Consume và luồng Outbound qua Virtual Threads. Nếu một Outbound Provider (như SMTP hay FCM) bị chậm, không được phép làm nghẹt luồng tiêu thụ của Kafka.
*   **Khả năng mở rộng (Scalability)**: Hỗ trợ scale-out nhiều Pods một cách trơn tru, không gây race-condition khi ghi nhận idempotency key, và định tuyến tin nhắn chính xác đến Pod giữ connection của user.

---

## 4. Ranh giới Triển khai (Scope Boundaries)

### 🟢 Nằm trong Phase 3 (In-Scope)
*   Lắng nghe và parsing CloudEvents v1.0 từ Kafka Broker.
*   Idempotency Guard ở cả tầng Application và DB.
*   Template Engine load `templates.yaml` và giải quyết Placeholder động (camelCase) cho ngôn ngữ tiếng Việt (`vi`).
*   Phân giải người nhận động (Dynamic Recipient) cho `BookingCancelled` (đối phương) và `DisputeResolved` (cả 2 bên).
*   Định tuyến thông minh dựa trên Redis Connection State Store tập trung.
*   Tích hợp Outbound Adapter thật bằng Spring Mail (Mailtrap) và Firebase Admin SDK.

### 🔴 Trì hoãn sang Phase sau (Out-Scope)
*   **Retry trì hoãn backoff nâng cao**: Cơ chế sử dụng scheduler để tự động thử lại sau 2s, 4s, 8s đối với lỗi Recoverable (như mạng chập chờn) sẽ được defer sang **Phase 5**. Trong Phase 3, nếu lỗi gửi tin, hệ thống ghi nhận attempt thất bại và không tự retry trì hoãn.
*   **REST API cho Inbox & Mark-Read**: API get danh sách thông báo và đánh dấu đã đọc thuộc về **Phase 4** (REST API).


---

## 4. Đặc tả Ngoại lệ & Lỗi (Edge Cases & Exception Handling)
*   **Lỗi Parsing JSON**: Nếu Event Payload bị hỏng, log error, đẩy vào Dead Letter Queue (DLQ) và ACK Kafka để không treo luồng tiêu thụ.
*   **Thiếu Required Fields**: Nếu Event thiếu field bắt buộc (Ví dụ: thiếu `bookingId` trong `BookingAccepted.v1`), log warning, không sinh thông báo và lưu audit log.
*   **Mất kết nối Redis/DB**: Sử dụng cơ chế Circuit Breaker hoặc Catch Exception để đảm bảo không bị mất Event. Kafka chỉ commit offset sau khi Event đã được xử lý và lưu DB thành công (At-least-once delivery).
