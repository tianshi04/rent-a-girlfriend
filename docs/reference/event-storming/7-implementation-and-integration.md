# TRIỂN KHAI VÀ TÍCH HỢP (IMPLEMENTATION & INTEGRATION)

Tài liệu này xác định kiến trúc tổng thể, các mẫu thiết kế (design patterns) mức hệ thống và chiến lược tích hợp nhằm chuyển đổi các Bounded Context đã được định nghĩa trong mô hình DDD thành các Microservices hoạt động độc lập, có thể mở rộng và bảo trì.

---

## 1. MAPPING BOUNDED CONTEXT SANG MICROSERVICES

Theo nguyên tắc "Một Bounded Context = Một Microservice", hệ thống **Rent-a-Girlfriend** sẽ được triển khai với các services sau:

1.  **Booking Service:** (Core) Chịu trách nhiệm quản lý vòng đời đặt lịch, tính toán thời gian và phối hợp các giao dịch SAGA cho luồng đặt/hủy/hoàn thành.
2.  **Finance Service:** (Core) Quản lý số dư ví nội bộ (Kano-Coin), cơ chế tạm giữ (Escrow) và chia sẻ hoa hồng nền tảng.
3.  **Profile & Catalogue Service:** (Core) Quản lý thông tin Companion, danh mục kịch bản dịch vụ (Scenario) và kiểm duyệt hình ảnh, âm thanh (Media).
4.  **Interaction Service:** (Supporting) Cung cấp hạ tầng liên lạc (Chat) theo thời gian thực và quản lý các đánh giá (Review).
5.  **Dispute Service:** (Supporting) Chứa bộ máy Workflow xử lý các khiếu nại và phán quyết từ phía Admin.
6.  **Identity Service:** (Generic) Xác thực, phân quyền (SSO/OAuth), quản lý rủi ro và trạng thái khóa/mở tài khoản.
7.  **Notification Service:** (Generic) Hạ tầng phân phối thông báo đa kênh, đóng gói logic về SSE (Server-Sent Events), FCM (Push) và Email.

---

## 2. KIẾN TRÚC TỔNG THỂ (MICROSERVICES ARCHITECTURE)

### 2.1. API Gateway & BFF (Backend for Frontend)
- Tất cả traffic từ các ứng dụng Client (Mobile/Web) sẽ không bao giờ gọi trực tiếp vào các microservices mà phải thông qua một **API Gateway**.
- API Gateway thực thi các vai trò kiểm soát ở viền hệ thống (Edge Level):
  - **Routing:** Định tuyến các request HTTP đến đúng Microservice tương ứng.
  - **Authentication:** Kiểm tra tính hợp lệ của Token từ Identity Service.
  - **Rate Limiting & CORS:** Bảo vệ hệ thống khỏi DDOS.
  - **BFF Pattern:** Có thể tổng hợp dữ liệu từ nhiều service để trả về một JSON gọn nhẹ nhất cho màn hình Mobile.

### 2.2. Cơ sở dữ liệu theo Service (Database-per-Service)
Để duy trì tính tự trị (Autonomy) và liên kết lỏng (Loose Coupling):
- **Tuyệt đối không sử dụng chung cơ sở dữ liệu vật lý.** Mỗi Microservice hoàn toàn sở hữu Database riêng. Nếu cần dữ liệu của service khác, bắt buộc phải gọi API hoặc lắng nghe Event.
- Lựa chọn hệ quản trị Database linh hoạt theo tính chất dữ liệu:
  - **Relational DB (PostgreSQL / MySQL):** Booking, Finance, Identity (Yêu cầu cao về ACID, tính toàn vẹn của Transaction).
  - **NoSQL (MongoDB):** Profile (Cần cấu trúc JSON động cho đa dạng Scenario và Media), Interaction (Tối ưu để ghi dữ liệu chat tốc độ cao).

---

## 3. CHIẾN LƯỢC GIAO TIẾP LIÊN DỊCH VỤ (INTER-SERVICE COMMUNICATION)

### 3.1. Giao tiếp Đồng bộ (Synchronous)
Sử dụng **RESTful API** hoặc **gRPC**. Phương pháp này chỉ áp dụng cho các hành vi **truy vấn** (Query) mà ở đó dữ liệu trả về là bắt buộc để có thể tiếp tục luồng xử lý hoặc không làm thay đổi trạng thái Aggregate của bên cấp dữ liệu (Open Host Service).
*   **Use-case 1:** `Booking Service` gọi `Profile Service` qua REST API để lấy *Snapshot* của một Scenario (thời lượng, giá tiền) ngay tại khoảnh khắc Client bấm đặt lịch.
*   **Use-case 2:** Các service nội bộ gọi `Identity Service` để lấy thông tin phân quyền của user.

### 3.2. Giao tiếp Bất đồng bộ (Asynchronous)
Sử dụng **Message Broker (RabbitMQ / Kafka)** làm nền tảng truyền tải **Domain Events**. Đây là phương thức giao tiếp chủ đạo để đảm bảo tính chịu lỗi và tránh "hiệu ứng sụp đổ chùm" (Cascading Failure).
*   Sử dụng mô hình Publish/Subscribe. Producer bắn Event và không cần chờ phản hồi, Consumer tự do subscribe theo nhu cầu.
*   **Use-case:** Khi `Booking Service` bắn event `BookingCompleted`, nó không cần biết service nào đang nhận. `Finance Service` tự bắt lấy để chia hoa hồng, `Interaction Service` bắt lấy để khóa khung chat, và `Notification Service` bắt để gửi Push.

---

## 4. QUẢN LÝ GIAO DỊCH PHÂN TÁN (DISTRIBUTED TRANSACTIONS)

Vì hệ thống không sử dụng database chung nên ta loại bỏ 2PC (Two-Phase Commit) và sử dụng **SAGA Pattern** để duy trì tính nhất quán cuối cùng (Eventual Consistency).

### 4.1. Các mô hình SAGA áp dụng
1.  **SAGA Orchestration (Điều phối tập trung):**
    *   Sử dụng ở các luồng phức tạp, có sự phụ thuộc thứ tự nghiêm ngặt hoặc cần Rollback (hoàn tác) tài chính.
    *   **Ví dụ:** Luồng *Booking Creation*. `Booking Service` là Orchestrator. Nó ra lệnh khóa tiền ở Finance, sau đó ra lệnh tạo Chat ở Interaction. Nếu Interaction lỗi mạng, nó ra lệnh Rollback hoàn tiền ở Finance.
    *   **Ví dụ:** Luồng *Dispute Resolution*. `Dispute Service` điều phối để chuyển tiền sau đó xóa bỏ đánh giá (Review).

2.  **SAGA Choreography (Tự điều phối phân tán):**
    *   Sử dụng ở các luồng đơn giản, khi các hành động nhánh là độc lập, không cần liên kết hủy bỏ nhau.
    *   **Ví dụ:** Luồng *Client Cancel Booking*. Dựa vào event của Booking, Finance tự xử lý luồng hoàn tiền, Interaction tự khóa phòng chat. Lỗi ở Interaction không bắt Finance thu hồi tiền lại.

### 4.2. Các Pattern Bảo đảm Độ tin cậy (Reliability Patterns)
Để kiến trúc Event-driven và SAGA không bị lỗi hệ thống, các Microservices bắt buộc tuân thủ:
*   **Transactional Outbox Pattern:** Khi Aggregate thay đổi trạng thái, Domain Event KHÔNG được gửi thẳng vào RabbitMQ. Sự kiện phải được lưu vào bảng `Outbox` cùng một Transaction Database (ACID) của nghiệp vụ. Sau đó có một Background Worker (ví dụ: Debezium) đọc bảng Outbox để đảm bảo Event được chuyển đi ít nhất một lần (At-Least-Once).
*   **Idempotency (Tính Lũy Đẳng):** Vì mạng có thể làm rớt tín hiệu trả về, Message Broker có thể gửi sự kiện đến 2 lần. Phía Consumer (`Finance Service`, `Interaction Service`) luôn lưu giữ một `Idempotency Key` (`sagaId` hoặc `eventId`). Nếu nhận lại event đã xử lý, trả về thành công ngay lập tức, bỏ qua thực thi nghiệp vụ lại.

---

## 5. TÍCH HỢP VỚI CÁC HỆ THỐNG BÊN NGOÀI (EXTERNAL INTEGRATION)

Đối với ranh giới tiếp xúc hệ thống ngoài (thường dùng Anti-Corruption Layer):
1.  **Tích hợp Payment (VNPay / Ví Điện Tử):** 
    *   `Finance Service` cung cấp IPN/Webhook Endpoint. 
    *   Logic nhận Webhook tách biệt, có xác thực `Signature` chữ ký và Job đối soát (Reconciliation) hàng ngày để phát hiện chênh lệch với VNPay.
2.  **Tích hợp Lưu trữ Media (AWS S3 / Cloudinary):**
    *   `Profile Service` không trực tiếp nhận luồng byte upload của User. 
    *   Áp dụng **Presigned URL Pattern**: Service chỉ cấp quyền upload một lần, Client tự tải trực tiếp lên Cloud. Sau khi có URL ảnh/voice, hệ thống lưu địa chỉ về MongoDB.
3.  **Tích hợp Thông báo (Firebase / SES):**
    *   `Notification Service` điều phối với third-party APIs. Áp dụng Fallback: cố gắng truyền qua kênh realtime (SSE), nếu báo lỗi thì đẩy sang kênh ngoại vi (FCM/Email).
