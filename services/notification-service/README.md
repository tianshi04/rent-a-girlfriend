# 🔔 NOTIFICATION SERVICE

**Phân loại Subdomain:** Generic Subdomain
**Mục tiêu:** Đóng gói toàn bộ hạ tầng gửi tin (SSE, FCM, Email), cung cấp cơ chế phân phối thông báo tập trung, tin cậy và không block các nghiệp vụ cốt lõi của hệ thống Rent-a-Girlfriend.

---

## 🏗️ CẤU TRÚC KIẾN TRÚC (HIGH-LEVEL ARCHITECTURE)

Dự án tuân thủ **Hexagonal Architecture**, được chia làm 4 lớp chính:

1. **Interfaces Layer**: `SseController` (Inbound Adapter nhận kết nối SSE), `Event Subscriber` (lắng nghe Kafka/RabbitMQ), `REST Controllers`.
2. **Application Layer**: `NotificationSubscriptionUseCase` (Inbound Port), `NotificationSubscriptionService` (UseCase implementation đẩy tin chưa đọc), `Routing Engine` (quyết định kênh gửi), `Notification Use Cases` (quản lý logic gửi tin).
3. **Domain Layer**: `Notification` (Aggregate Root) và `DeliveryAttempt` (Entity), chứa các luật kinh doanh (ví dụ: Retry tối đa 3 lần).
4. **Infrastructure Layer**: `SseConnectionRegistry` (Quản lý session Emitter cục bộ), `RedisPubSubAdapter` (Hạ tầng truyền tin distributed), `FCM Adapter` (Firebase), `SMTP Adapter` (Email), `PostgreSQL Repository` (DB).

---

## 📚 TÀI LIỆU KỸ THUẬT (DOCUMENTATION INDEX)

- **[01. Taxonomy & Delivery Policy](./docs/notification-taxonomy.md)**: Định nghĩa các loại thông báo (Transactional, Interaction, Promotional) và chính sách phân phối (Priority, Realtime, Push, Retry, Persistence).
- **[02. Routing Policy](./docs/routing-policy.md)**: Định nghĩa luật định tuyến gửi thông báo qua kênh nào (SSE, FCM, hay Email) dựa vào trạng thái kết nối và mức độ ưu tiên.
- **[03. Domain Model](./docs/notification-domain-model.md)**: Thiết kế thực thể `Notification` và `DeliveryAttempt` chuẩn DDD.
- **[04. Event Catalog](./docs/event-catalog.md)**: Định nghĩa Contract cho Inbound Events (Hybrid) và Outbound Events.
- **[05. Domain Event Mapping](./docs/domain-event-mapping.md)**: Chi tiết cách ánh xạ từ Domain Events sang nội dung thông báo.
- **[06. 📡 Event Integration Guide](./docs/event-integration-guide.md)**: **[ĐỐI NGOẠI]** Hướng dẫn cho các team khác (Booking, Finance, Chat, Profile, Identity, Dispute) biết cần publish event gì để Notification Service hoạt động.
- **[07. System Architecture & Flow](./docs/architecture.md)**: Sơ đồ kiến trúc xử lý tin nhắn phân tán (Distributed SSE) và luồng dữ liệu (Data Flow) cốt lõi của Notification Service.
- **[08. State Machine](./docs/state-machine.md)**: Định nghĩa vòng đời (Lifecycle), sơ đồ chuyển đổi trạng thái (State Diagram) và cơ chế xử lý lỗi (Failure Handling).
- **[09. Realtime Delivery (SSE)](./docs/realtime-delivery.md)**: Hợp đồng giao tiếp (Contract) giữa Client và Server cho luồng Server-Sent Events, tích hợp Istio Auth.
- **[10. API Contract](./docs/api-contract.md)**: Đặc tả danh sách API RESTful (Inbox, Mark as read) cho Frontend và Mobile.
- **[11. Data Model](./docs/data-model.md)**: Thiết kế sơ đồ thực thể (Entity Relationship) và cấu trúc bảng trong PostgreSQL.
- **[12. Notification Templates](./config/templates.yaml)**: File cấu hình quản lý nội dung thông báo tập trung (19 event types, vi/en).
- **[ADR-0001: Scope & Goals](./docs/adr/0001-notification-service-scope-and-goals.md)**: Định nghĩa ranh giới Bounded Context của service, những gì In-Scope và Out-Scope.
- **[ADR-0002: Database Choice](./docs/adr/0002-database-choice-postgresql.md)**: Quyết định sử dụng PostgreSQL để quản lý lịch sử gửi lỗi (Audit & Traceability).
- **[ADR-0003: SSE Authentication Strategy](./docs/adr/0003-sse-authentication-strategy.md)**: Quyết định cấm truyền JWT qua URL và bắt buộc dùng Header qua thư viện ngoài cho Frontend.
- **[ADR-0004: Cursor-based Pagination](./docs/adr/0004-cursor-based-pagination-for-inbox.md)**: Quyết định sử dụng Cursor-based pagination thay vì Page-based cho API Inbox để tránh lỗi lặp data trong môi trường realtime.
- **[ADR-0005: Hybrid Triggering](./docs/adr/0005-hybrid-notification-triggering-strategy.md)**: Chiến lược kết hợp giữa Smart Consumer (nghe Domain Events) và Passive Subscriber.
- **[ADR-0006: Payload Design Strategy](./docs/adr/0006-payload-design-operational-flexibility-vs-type-safety.md)**: Quyết định lựa chọn Map (Operational Flexibility) thay vì Class Hierarchy (Type Safety) để tối ưu hóa sự tiến hóa liên tục của payload.
- **[ADR-0007: Outbound Delivery & Error Handling](./docs/adr/0007-outbound-delivery-and-error-handling-strategy.md)**: Quy định về SendResult, Retry Policy riêng cho từng kênh gửi và cơ chế Async Queue / Worker Pool bất đồng bộ.
- **[ADR-0010: Refactoring SSE Connection Management](./docs/adr/phase-2/0010-sse-clean-architecture-refactoring.md)**: Quyết định kiến trúc cô lập `SseEmitter` ra khỏi Core và thiết lập UseCase độc lập.
- **[13. Ports & Adapters Architecture Levels (SSE)](./docs/todo/phase-2/ports-and-adapters-sse.md)**: Tài liệu đặc tả 3 cấp độ (Level) kiến trúc Ports & Adapters sạch cho case SSE.

---

## 📈 BÁO CÁO TIẾN ĐỘ (DEVELOPMENT PROGRESS REPORTS)

Các báo cáo tổng hợp tiến độ và đánh giá kỹ thuật của dự án theo dòng thời gian:
- **[📅 Báo Cáo Tiến Độ 22-05-2026](./docs/time-line/22-05-2026.md)**: Đánh giá chi tiết hoàn thành Phase 0 & Phase 1, phát hiện các lỗi thiết kế Database UNIQUE constraint, Exception mapping và đề xuất lộ trình Phase 2 (SSE) & Phase 4 (REST API).

---

## 🧪 KIỂM THỬ KHÓI (SMOKE TESTING)

Dịch vụ hỗ trợ một bộ Smoke Test chạy trong môi trường Docker Compose cô lập, tự động khởi động cơ sở dữ liệu PostgreSQL và Notification Service, sau đó thực hiện gọi kiểm tra Actuator Health Check để xác nhận dịch vụ khởi động hoàn toàn ổn định (`UP`).

Để khởi chạy Smoke Test trên Windows:
```powershell
cd tests/smoke
powershell -ExecutionPolicy Bypass -File .\smoke-test.ps1
```

Script sẽ tự động build image, cấu hình kết nối DB qua `DB_URL` khớp với cấu hình trong `application.yml`, kiểm tra trạng thái và dọn dẹp môi trường sau khi hoàn tất.



