# Dispute Resolution Service

Microservice hỗ trợ giải quyết tranh chấp (Dispute Resolution Supporting Subdomain) giữa Client và Companion khi có sự cố phát sinh (ví dụ: vắng mặt, lừa đảo, hành vi không đúng mực) trong nền tảng Rent-a-Girlfriend. Service tích hợp quy trình điều phối giao dịch phân tán SAGA đáng tin cậy cao cùng cơ chế Transactional Outbox.

---

## Vai trò cốt lõi của Service

Dispute Resolution Service được xây dựng theo kiến trúc Hexagonal (Ports & Adapters) với phân tách CQRS rõ rệt thông qua các cổng giao tiếp REST và gRPC:

### 1. Dispute Lifecycle & Commands (Ghi/Thay đổi trạng thái)
- **Tạo khiếu nại (Create Report)**: Client hoặc Companion tạo khiếu nại với lý do cụ thể và minh chứng bằng chứng (Evidence) dạng văn bản hoặc hình ảnh.
- **Phân công Admin (Assign Admin)**: Admin tự nhận hoặc được hệ thống phân phối để xử lý tranh chấp, đưa trạng thái Dispute từ `OPEN` sang `RESOLVING`.
- **Giải quyết tranh chấp (Resolve Dispute)**: Admin đưa ra quyết định cuối cùng với ba hình thức xử lý chính:
  - `REFUND_CLIENT`: Hoàn tiền cọc từ ví Escrow về ví Client. Kích hoạt SAGA Refund.
  - `PAYOUT_COMPANION`: Quyết toán tiền từ ví Escrow cho Companion. Kích hoạt SAGA Payout.
  - `REJECT`: Bác bỏ khiếu nại, không có giao dịch tài chính xảy ra.

### 2. Phân tán SAGA Orchestration (Quy trình hoàn tiền/Quyết toán)
Được quản lý thông qua hai Orchestrator với quy tắc tự phục hồi và khả năng chịu lỗi tối đa:
- **Refund SAGA Flow**:
  1. `REFUNDING`: Gọi cổng Finance Service thực hiện hoàn trả tiền cọc.
  2. `HIDING_REVIEW`: Gọi cổng Interaction Service để ẩn đánh giá xấu của booking đó và khóa phòng chat liên quan.
- **Payout SAGA Flow**:
  1. `PAYING_OUT`: Gọi cổng Finance Service để trả tiền cho Companion.
  2. `LOCKING_CHAT`: Gọi cổng Interaction Service để khóa phòng chat booking.
- **Cơ chế chịu lỗi nghiệp vụ**:
  - Giao dịch hoàn tiền/quyết toán không thể rollback nếu bước Interaction sau đó thất bại.
  - Áp dụng chiến lược **Infinite Retry (Thử lại vô hạn)** tại các bước liên lạc với Interaction Service, đảm bảo cuối cùng dữ liệu sẽ nhất quán (Eventual Consistency) mà không bao giờ bị mất hoặc rollback nhầm tiền.

### 3. REST Queries & Admin Dashboard (Đọc/Tra cứu)
- **List Disputes (REST)**: Tra cứu toàn bộ tranh chấp, hỗ trợ lọc theo trạng thái (`status`) và phân trang.
- **Dispute Detail (REST)**: Xem chi tiết nội dung khiếu nại và toàn bộ bằng chứng liên quan.
- **SAGA State Viewer (REST)**: Theo dõi tiến độ chạy, số lần retry và lỗi gần nhất của các quy trình giao dịch SAGA.

---

## Business Invariants & Rules

Service chịu trách nhiệm thực thi nghiêm ngặt các quy tắc nghiệp vụ bất biến sau:
- `[INV-D01]` Mỗi Booking ID chỉ được tồn tại tối đa **một** khiếu nại đang hoạt động (`OPEN` hoặc `RESOLVING`).
- `[INV-D02]` Khi Dispute đã chuyển sang trạng thái cuối (`REFUNDED`, `PAID_OUT`, `REJECTED`), không một Admin nào được phép thay đổi thông tin hay giải quyết lại.
- **Lý do hợp lệ (VALID_REASONS)**: Chỉ chấp nhận các lý do khiếu nại nằm trong tập hợp chuẩn hóa: `NO_SHOW` (Không đến hẹn), `FRAUD` (Lừa đảo), `MISCONDUCT` (Hành vi không đúng mực), và `OTHER` (Lý do khác).

---

## Tech Stack

- **Language**: Python 3.12 (Quản lý qua `uv` CLI)
- **REST Framework**: FastAPI + Uvicorn (Phục vụ Dashboard Queries)
- **gRPC Framework**: `grpcio` + Protobuf (Phục vụ Commands giải quyết tranh chấp)
- **Database**: PostgreSQL (SQLAlchemy Async ORM) / SQLite in-memory phục vụ môi trường test.
- **Event Broker**: `aiokafka` (Kafka Consumer cho Saga replies và Transactional Outbox Worker)
- **Testing**: `pytest` + `pytest-asyncio` + `aiosqlite` (Chia sẻ connection qua StaticPool)

---

## Quick Start

### 1. Thiết lập môi trường

```bash
# Copy cấu hình môi trường mẫu
cp .env.example .env

# Cài đặt dependencies và khởi tạo môi trường ảo qua uv
uv sync
```

### 2. Sinh mã Protobuf (Nếu thay đổi hợp đồng)

```bash
make generate-proto
```

### 3. Chạy Service

Service chạy đồng thời FastAPI web server (port `8082`), gRPC server (port `50051`), Outbox Worker, Saga Retry Worker, và Kafka Consumer:

```bash
make run
```

---

## API Endpoints

### REST API (Queries & Dashboard)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/api/v1/health` | Health Check | Public |
| GET | `/api/v1/disputes` | List disputes (phân trang + lọc status) | ADMIN |
| GET | `/api/v1/disputes/{id}` | Chi tiết khiếu nại | ADMIN |
| GET | `/api/v1/disputes/{id}/saga` | Lấy trạng thái quy trình giao dịch SAGA | ADMIN |

### gRPC API (Commands)

| RPC | Service | Description | Role |
|-----|---------|-------------|------|
| `CreateReport` | `DisputeService` | Khởi tạo khiếu nại | Client / Companion |
| `ResolveDispute` | `DisputeService` | Đóng khiếu nại & kích hoạt SAGA tài chính | Admin |

---

## Testing

Chạy toàn bộ suite kiểm thử gồm 16 Unit Tests (Domain logic) và 15 Integration Tests (FastAPI, gRPC, DB, SAGA Flows):

```bash
make test
```
