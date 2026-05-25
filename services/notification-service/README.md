# 🔔 NOTIFICATION SERVICE

**Phân loại Subdomain:** Generic Subdomain  
**Mục tiêu:** Đóng gói toàn bộ hạ tầng gửi tin (SSE, FCM, Email), cung cấp cơ chế phân phối thông báo tập trung, tin cậy và không block các nghiệp vụ cốt lõi của hệ thống Rent-a-Girlfriend.

---

## 🚀 HƯỚNG DẪN CHẠY (GETTING STARTED)

### Yêu cầu môi trường

| Công cụ     | Phiên bản tối thiểu | Ghi chú                          |
| :---------- | :------------------ | :------------------------------- |
| Java (JDK)  | 21                  | Bắt buộc để build JAR cục bộ     |
| Docker      | 24+                 | Bắt buộc để chạy containers      |
| Docker Compose | v2+              | Tích hợp sẵn trong Docker Desktop|
| make        | Bất kỳ              | Tùy chọn (Windows: cài qua Choco)|

### Bước 1: Cấu hình biến môi trường

Sao chép file mẫu và điền thông tin thực tế:
```bash
cp .env.example .env
```

Chỉnh sửa file `.env` với thông tin kết nối thực tế của bạn:
```env
# Database (Neon Cloud PostgreSQL)
DB_URL=jdbc:postgresql://<neon-host>/neondb?sslmode=require
DB_USERNAME=<neon-user>
DB_PASSWORD=<neon-password>

# SMTP (Mailtrap Sandbox)
SMTP_USERNAME=<mailtrap-user>
SMTP_PASSWORD=<mailtrap-password>
```

> [!NOTE]
> Redis và Kafka sẽ được khởi chạy tự động bên trong Docker Compose.
> Bạn **không** cần cài sẵn Redis hay Kafka trên máy host.

---

### Cách 1: Clone về & chạy ngay (Khuyên dùng cho lần đầu hoặc CI/CD)

Không cần cài Java. Docker sẽ tự build mọi thứ bên trong container.

```bash
make up
```

Hoặc nếu không dùng `make`:
```bash
docker compose up --build -d
```

> [!IMPORTANT]
> Lần đầu sẽ mất **3-5 phút** để Docker tải image JDK và build JAR bên trong container.
> Từ lần thứ 2 trở đi, các layer sẽ được cache lại và nhanh hơn đáng kể.

---

### Cách 2: Dev cục bộ (Siêu nhanh, khuyên dùng khi đang phát triển)

Yêu cầu Java 21 đã được cài trên máy host.

```bash
make dev
```

Hoặc nếu không dùng `make`:
```bash
# Bước 1: Build file JAR trên máy host (2-5 giây nhờ Gradle Daemon cache)
./gradlew bootJar          # Linux/Mac
.\gradlew.bat bootJar      # Windows

# Bước 2: Build image Docker từ file JAR đã có và khởi chạy (< 2 giây)
docker compose -f docker-compose.local.yml up --build -d
```

> [!TIP]
> Khi sửa code, chỉ cần lặp lại 2 bước trên. Toàn bộ quy trình mất **dưới 10 giây**.

---

### Kiểm tra dịch vụ đã chạy thành công

```bash
curl http://localhost:8084/actuator/health
```

Kết quả mong đợi:
```json
{
  "status": "UP",
  "components": {
    "db":    { "status": "UP" },
    "redis": { "status": "UP" },
    "ping":  { "status": "UP" }
  }
}
```

---

### Dừng dịch vụ

```bash
make down
```

Hoặc:
```bash
docker compose -f docker-compose.local.yml down   # nếu dùng Cách 2
docker compose down                                # nếu dùng Cách 1
```

---

### Tổng hợp các lệnh `make`

| Lệnh           | Mô tả                                                             |
| :------------- | :---------------------------------------------------------------- |
| `make up`      | **[Clone/CI-CD]** Build & chạy toàn bộ trong Docker (tự build JAR trong container) |
| `make dev`     | **[Local Dev]** Build JAR trên host rồi chạy container (siêu nhanh, < 10s) |
| `make down`    | Dừng toàn bộ Docker containers                                   |
| `make build`   | Chỉ build file JAR (không chạy Docker)                           |
| `make run`     | Chạy ứng dụng trực tiếp trên máy host (không cần Docker)         |
| `make test-unit` | Chạy toàn bộ bộ kiểm thử                                      |
| `make clean`   | Xóa toàn bộ artifact đã build                                    |
| `make docker-build` | Build standalone Docker image gán tag `latest`             |
| `make test-docker` | Tất cả test             |
| `make test-docker-unit` | Chỉ @Tag("unit")             |
| `make test-docker-int` | Chỉ @Tag("integration")         |

---

## 🏗️ CẤU TRÚC KIẾN TRÚC (HIGH-LEVEL ARCHITECTURE)

Dự án tuân thủ **Hexagonal Architecture**, được chia làm 4 lớp chính:

1. **Interfaces Layer**: `SseController` (Inbound Adapter nhận kết nối SSE), `Event Subscriber` (lắng nghe Kafka), `REST Controllers`.
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
- **[ADR-0001: Scope & Goals](./docs/adr/0001-notification-service-scope-and-goals.md)**: Định nghĩa ranh giới Bounded Context của service.
- **[ADR-0002: Database Choice](./docs/adr/0002-database-choice-postgresql.md)**: Quyết định sử dụng PostgreSQL.
- **[ADR-0003: SSE Authentication Strategy](./docs/adr/0003-sse-authentication-strategy.md)**: Quyết định cấm truyền JWT qua URL.
- **[ADR-0004: Cursor-based Pagination](./docs/adr/0004-cursor-based-pagination-for-inbox.md)**: Quyết định sử dụng Cursor-based pagination.
- **[ADR-0005: Hybrid Triggering](./docs/adr/0005-hybrid-notification-triggering-strategy.md)**: Chiến lược kết hợp Smart Consumer và Passive Subscriber.
- **[ADR-0006: Payload Design Strategy](./docs/adr/0006-payload-design-operational-flexibility-vs-type-safety.md)**: Quyết định dùng Map thay vì Class Hierarchy.
- **[ADR-0007: Outbound Delivery & Error Handling](./docs/adr/0007-outbound-delivery-and-error-handling-strategy.md)**: Quy định về SendResult, Retry Policy và Async Queue.
- **[ADR-0010: Refactoring SSE Connection Management](./docs/adr/phase-2/0010-sse-clean-architecture-refactoring.md)**: Quyết định kiến trúc cô lập `SseEmitter`.

---

## 📈 BÁO CÁO TIẾN ĐỘ (DEVELOPMENT PROGRESS REPORTS)

- **[📅 Báo Cáo Tiến Độ 22-05-2026](./docs/time-line/22-05-2026.md)**: Đánh giá chi tiết hoàn thành Phase 0 & Phase 1, phát hiện các lỗi thiết kế và đề xuất lộ trình Phase 2 (SSE) & Phase 4 (REST API).

---

## 🧪 KIỂM THỬ

```bash
make test-unit
```

Hoặc:
```bash
./gradlew test
```

Xem mục lục kiểm thử chi tiết tại [TESTS_README.md](./tests/TESTS_README.md).
