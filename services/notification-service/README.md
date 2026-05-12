# 🔔 NOTIFICATION SERVICE

**Phân loại Subdomain:** Generic Subdomain
**Mục tiêu:** Đóng gói toàn bộ hạ tầng gửi tin (SSE, FCM, Email), cung cấp cơ chế phân phối thông báo tập trung, tin cậy và không block các nghiệp vụ cốt lõi của hệ thống Rent-a-Girlfriend.

---

## 🏗️ CẤU TRÚC KIẾN TRÚC (HIGH-LEVEL ARCHITECTURE)

Dự án tuân thủ **Hexagonal Architecture**, được chia làm 4 lớp chính:

1. **Interfaces Layer**: `Event Subscriber` (lắng nghe Kafka/RabbitMQ), `REST Controllers` (nếu cần query nội bộ).
2. **Application Layer**: `Routing Engine` (quyết định kênh gửi), `Notification Use Cases` (quản lý logic gửi tin).
3. **Domain Layer**: `Notification` (Aggregate Root) và `DeliveryAttempt` (Entity), chứa các luật kinh doanh (ví dụ: Retry tối đa 3 lần).
4. **Infrastructure Layer**: `FCM Adapter` (Firebase), `SSE Manager` (Socket), `SMTP Adapter` (Email), `PostgreSQL Repository` (DB).

---

## 📚 TÀI LIỆU KỸ THUẬT (DOCUMENTATION INDEX)

- **[01. Taxonomy & Delivery Policy](./docs/notification-taxonomy.md)**: Định nghĩa các loại thông báo (Transactional, Interaction, Promotional) và chính sách phân phối (Priority, Realtime, Push, Retry, Persistence).
- **[02. Routing Policy](./docs/routing-policy.md)**: Định nghĩa luật định tuyến gửi thông báo qua kênh nào (SSE, FCM, hay Email) dựa vào trạng thái kết nối và mức độ ưu tiên.
- **[03. Domain Model](./docs/notification-domain-model.md)**: Thiết kế thực thể `Notification` và `DeliveryAttempt` chuẩn DDD.
- **[04. Event Catalog](./docs/event-catalog.md)**: Định nghĩa Contract cho Inbound Events (Hybrid) và Outbound Events.
- **[05. Domain Event Mapping](./docs/domain-event-mapping.md)**: Chi tiết cách ánh xạ từ Domain Events sang nội dung thông báo.
- **[06. System Architecture & Flow](./docs/architecture.md)**: Sơ đồ kiến trúc xử lý tin nhắn phân tán (Distributed SSE) và luồng dữ liệu (Data Flow) cốt lõi của Notification Service.
- **[07. State Machine](./docs/state-machine.md)**: Định nghĩa vòng đời (Lifecycle), sơ đồ chuyển đổi trạng thái (State Diagram) và cơ chế xử lý lỗi (Failure Handling).
- **[08. Realtime Delivery (SSE)](./docs/realtime-delivery.md)**: Hợp đồng giao tiếp (Contract) giữa Client và Server cho luồng Server-Sent Events, tích hợp Istio Auth.
- **[09. API Contract](./docs/api-contract.md)**: Đặc tả danh sách API RESTful (Inbox, Mark as read) cho Frontend và Mobile.
- **[10. Data Model](./docs/data-model.md)**: Thiết kế sơ đồ thực thể (Entity Relationship) và cấu trúc bảng trong PostgreSQL.
- **[11. Notification Templates](./config/templates.yaml)**: File cấu hình quản lý nội dung thông báo tập trung.
- **[ADR-0001: Scope & Goals](./docs/adr/0001-notification-service-scope-and-goals.md)**: Định nghĩa ranh giới Bounded Context của service, những gì In-Scope và Out-Scope.
- **[ADR-0002: Database Choice](./docs/adr/0002-database-choice-postgresql.md)**: Quyết định sử dụng PostgreSQL để quản lý lịch sử gửi lỗi (Audit & Traceability).
- **[ADR-0003: SSE Authentication Strategy](./docs/adr/0003-sse-authentication-strategy.md)**: Quyết định cấm truyền JWT qua URL và bắt buộc dùng Header qua thư viện ngoài cho Frontend.
- **[ADR-0004: Cursor-based Pagination](./docs/adr/0004-cursor-based-pagination-for-inbox.md)**: Quyết định sử dụng Cursor-based pagination thay vì Page-based cho API Inbox để tránh lỗi lặp data trong môi trường realtime.
- **[ADR-0005: Hybrid Triggering](./docs/adr/0005-hybrid-notification-triggering-strategy.md)**: Chiến lược kết hợp giữa Smart Consumer (nghe Domain Events) và Passive Subscriber.
