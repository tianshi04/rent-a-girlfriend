# Infrastructure — Finance Service

Tài liệu mô tả chi tiết tầng hạ tầng: Database Schema, Outbox Pattern, VNPay Adapter và Kafka Integration.

---

## Database Schema (PostgreSQL)

Finance Service sở hữu schema riêng biệt, không chia sẻ với bất kỳ service nào khác (**Zero Cross-Service DB Access**).

**Connection string:** `postgresql+asyncpg://<user>:<password>@<host>:<port>/finance_service`

### Bảng `wallets`

| Column | Type | Constraint | Mô tả |
|---|---|---|---|
| `wallet_id` | `VARCHAR(36)` | PRIMARY KEY | UUID |
| `user_id` | `VARCHAR(36)` | UNIQUE, NOT NULL, INDEX | UUID của chủ sở hữu |
| `available_balance` | `INTEGER` | DEFAULT 0, NOT NULL | Số Kano-Coin có thể dùng |
| `frozen_balance` | `INTEGER` | DEFAULT 0, NOT NULL | Số Kano-Coin đang bị khóa |
| `created_at` | `DATETIME` | DEFAULT utcnow | — |
| `updated_at` | `DATETIME` | DEFAULT utcnow, ON UPDATE | — |

### Bảng `escrows`

| Column | Type | Constraint | Mô tả |
|---|---|---|---|
| `escrow_id` | `VARCHAR(36)` | PRIMARY KEY | UUID |
| `booking_id` | `VARCHAR(36)` | UNIQUE, NOT NULL, INDEX | UUID của Booking |
| `amount` | `INTEGER` | NOT NULL | Số Kano-Coin đang giữ |
| `status` | `VARCHAR(20)` | DEFAULT 'HELD', INDEX | `HELD`, `PAID_OUT`, `REFUNDED` |
| `created_at` | `DATETIME` | — | — |
| `updated_at` | `DATETIME` | ON UPDATE | — |

### Bảng `transactions`

| Column | Type | Constraint | Mô tả |
|---|---|---|---|
| `transaction_id` | `VARCHAR(36)` | PRIMARY KEY | UUID |
| `user_id` | `VARCHAR(36)` | NOT NULL, INDEX | UUID |
| `amount` | `INTEGER` | NOT NULL | Số Kano-Coin |
| `type` | `VARCHAR(30)` | NOT NULL, INDEX | `TOPUP`, `BOOKING_RESERVATION`, `ESCROW_RELEASE`, `REFUND` |
| `status` | `VARCHAR(20)` | NOT NULL, INDEX | `PENDING`, `SUCCESS`, `FAILED` |
| `reference_id` | `VARCHAR(36)` | NOT NULL, INDEX | VNPay `txn_id` hoặc `booking_id` |
| `created_at` | `DATETIME` | — | — |

> **Khuyến nghị Performance:** Tạo **Composite Index** cho cặp `(reference_id, type)` để tăng tốc truy vấn idempotency check.
> ```sql
> CREATE INDEX idx_txn_ref_type ON transactions (reference_id, type);
> ```

### Bảng `outbox`

| Column | Type | Constraint | Mô tả |
|---|---|---|---|
| `id` | `INTEGER` | PRIMARY KEY, AUTOINCREMENT | — |
| `event_id` | `VARCHAR(36)` | UNIQUE, NOT NULL | UUID ngẫu nhiên |
| `event_type` | `VARCHAR(200)` | NOT NULL | Ví dụ: `finance.wallet-topped-up.v1` |
| `payload` | `TEXT` | NOT NULL | JSON string (Protobuf MessageToDict) |
| `created_at` | `DATETIME` | — | — |
| `processed` | `BOOLEAN` | DEFAULT false, INDEX | Đánh dấu đã gửi lên Kafka |

---

## Transactional Outbox Pattern

**File:** [`internal/infrastructure/broker/outbox_publisher.py`](../internal/infrastructure/broker/outbox_publisher.py)

### Vấn đề giải quyết

Khi một command thay đổi state DB (ví dụ: credit ví) và cần publish event lên Kafka, có thể xảy ra tình trạng DB commit thành công nhưng Kafka gửi thất bại (hoặc ngược lại) — dẫn đến mất nhất quán.

### Giải pháp

1. **`DatabaseEventPublisher.publish(event)`** — Trong cùng một DB transaction với command, insert bản ghi vào bảng `outbox`. Không gửi Kafka trực tiếp.
2. **`OutboxPublisherWorker`** — Background task polling bảng `outbox` mỗi `500ms`, gửi các bản ghi chưa processed lên Kafka, đánh dấu `processed = True`.

### Đảm bảo

- **At-Least-Once Delivery:** Nếu Worker crash sau khi gửi nhưng trước khi đánh dấu `processed`, event sẽ được gửi lại. Consumer phải xử lý idempotent.
- **Ordering:** Events được gửi theo `created_at ASC`. Partition key là `booking_id` hoặc `user_id`.

### CloudEvents Format (Kafka message)

```json
{
  "specversion": "1.0",
  "id": "<event_id>",
  "source": "/rent-a-gf/finance-service/<user_id>",
  "type": "finance.wallet-topped-up.v1",
  "datacontenttype": "application/json",
  "time": "2024-01-01T00:00:00Z",
  "data": { ... },
  "extensions": {
    "correlationId": "<event_id>"
  }
}
```

---

## VNPay Adapter

**File:** [`internal/infrastructure/payment/vnpay.py`](../internal/infrastructure/payment/vnpay.py)

### Cấu hình

| Setting | Env var | Mô tả |
|---|---|---|
| TMN Code | `VNPAY_TMN_CODE` | Merchant code từ VNPay portal |
| Hash Secret | `VNPAY_HASH_SECRET` | HMAC-SHA512 secret key |
| Payment URL | `VNPAY_URL` | `https://sandbox.vnpayment.vn/paymentv2/vpcpay.html` |
| Return URL | `VNPAY_RETURN_URL` | Endpoint nhận redirect sau thanh toán |

### `generate_payment_url(txn_ref, amount_vnd, ip_address)`

Sinh URL thanh toán có chữ ký HMAC-SHA512. Tất cả tham số `vnp_*` được sắp xếp alphabetical và ký trước khi encode.

### `validate_ipn_signature(params)`

Tách `vnp_SecureHash` ra khỏi params, tái tạo chuỗi ký, so sánh HMAC. Trả `True`/`False`.

### Luồng thanh toán đầy đủ

```
Client App                Finance Service           VNPay
   │                           │                      │
   │── POST /topup ────────────▶│                      │
   │                           │─── generate_url() ──▶│
   │◀── { payment_url } ───────│                      │
   │                           │                      │
   │── redirect to VNPay ──────────────────────────────▶│
   │                           │                      │ (User pays)
   │                           │◀─── GET /vnpay-ipn ──│ (IPN server-to-server)
   │                           │─── validate_sig() ───│
   │                           │─── credit wallet ────│
   │                           │─── return RspCode 00 ▶│
   │◀─────────────── redirect GET /vnpay-return ───────│
```

---

## Kafka Integration

### Producer (Outbox Worker)

- **Broker config:** `KAFKA_BROKERS` (mặc định `localhost:9092`)
- **Topic:** `finance.events`
- **Serializer:** JSON (`AIOKafkaProducer` với `value_serializer`)
- **Partition key:** `booking_id` hoặc `user_id` (đảm bảo ordering trong partition)

### Consumer (Identity Event Listener)

- **Topic:** `identity.events`
- **Consumer Group:** `finance-service-onboarder`
- **Mục đích:** Lắng nghe `UserRegistered` event để tự động khởi tạo ví mới (Option B — Event-Driven Onboarding)
- **Retry:** 5 lần retry với 5s delay nếu Kafka không sẵn sàng
- **Idempotency:** `create_wallet_onboard()` kiểm tra ví đã tồn tại trước khi tạo mới

### Idempotency cho Consumer

Finance Service xử lý idempotency theo hai cơ chế:
1. **Wallet onboarding:** Kiểm tra `find_by_user_id()` trước khi tạo ví mới
2. **VNPay IPN:** Kiểm tra `transaction.status != PENDING` — reject nếu đã xử lý

---

## Dependency Injection (Bootstrap)

**File:** [`internal/bootstrap/__init__.py`](../internal/bootstrap/__init__.py)

```
FastAPI Request
    │
    ├── Depends(get_db_session)        → AsyncSession (auto-commit / auto-rollback)
    │       │
    │       └── Depends(get_finance_cmd) → FinanceCommandService
    │                   ├── WalletRepository(session)
    │                   ├── EscrowRepository(session)
    │                   ├── TransactionRepository(session)
    │                   ├── DatabaseEventPublisher(session)
    │                   └── VNPayAdapter (singleton)
    │
    └── Response
```

**`get_db_session`** tự động commit sau mỗi request thành công, rollback nếu có exception.
