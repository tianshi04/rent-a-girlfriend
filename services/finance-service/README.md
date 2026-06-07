# Finance Service

Dịch vụ Tài chính (Finance Service) phụ trách quản lý **Ví Kano-Coin (Wallet)**, **Quỹ giữ tiền trung gian (Escrow)**, **Nhật ký giao dịch (Ledger)** và tích hợp Cổng thanh toán **VNPay Sandbox**.

Tuân thủ kiến trúc **Hexagonal Architecture** + **Domain-Driven Design (DDD)** + **Transactional Outbox Pattern**.

---

## 📚 Tài liệu

| File | Nội dung |
|---|---|
| [`docs/01_domain.md`](docs/01_domain.md) | Aggregates, Value Objects, Invariants, Events, Domain Services |
| [`docs/02_api.md`](docs/02_api.md) | REST API & gRPC Contract Reference |
| [`docs/03_infrastructure.md`](docs/03_infrastructure.md) | DB Schema, Outbox Pattern, VNPay, Kafka |
| [`docs/04_testing.md`](docs/04_testing.md) | Test strategy, 35 test cases, cách thêm test mới |

---

## 🚀 Khởi chạy nhanh (WSL / Linux)

### 1. Cài đặt dependencies

```bash
cd services/finance-service
uv sync
```

### 2. Biên dịch Protobuf contracts

```bash
make generate-proto
```

### 3. Chạy test suite

```bash
make test
```

**Kết quả kỳ vọng:** `35 passed`

### 4. Cấu hình môi trường

```bash
cp .env.example .env
# Chỉnh sửa .env: DB_HOST, KAFKA_BROKERS, VNPAY_TMN_CODE, VNPAY_HASH_SECRET...
```

### 5. Khởi chạy service

```bash
make run
```

Service khởi động đồng thời:
- **FastAPI REST** trên cổng `8080`
- **gRPC Server** trên cổng `50051`
- **Outbox Worker** (background polling → Kafka `finance.events`)
- **Identity Event Listener** (Kafka consumer `identity.events` → auto onboard wallet)

---

## 🏗️ Cấu trúc dự án

```
services/finance-service/
├── docs/
│   ├── 01_domain.md            # DDD: Aggregates, Invariants, Events
│   ├── 02_api.md               # REST & gRPC API Reference
│   ├── 03_infrastructure.md    # DB Schema, Outbox, VNPay, Kafka
│   └── 04_testing.md           # Test strategy & coverage
├── scripts/
│   └── generate_proto.py       # Compile Protobuf → gen/
├── internal/
│   ├── bootstrap/
│   │   └── __init__.py         # DI container, DB engine, VNPay adapter, Outbox worker
│   ├── domain/
│   │   ├── aggregate/
│   │   │   ├── wallet.py       # Wallet: [INV-F01, F02, F03]
│   │   │   ├── escrow.py       # Escrow: [INV-F04, F05] + commission
│   │   │   └── transaction.py  # Transaction: Audit Ledger
│   │   ├── events.py           # Domain Events (immutable dataclasses)
│   │   ├── service.py          # CommissionCalculator, CurrencyExchange
│   │   ├── vo.py               # Money Value Object
│   │   ├── errors.py           # Domain exceptions
│   │   └── repository.py       # Repository interfaces
│   ├── application/
│   │   ├── command/
│   │   │   └── finance.py      # Use Cases: Freeze, Escrow, Payout, Refund, VNPay IPN
│   │   └── port.py             # IEventPublisher interface
│   ├── infrastructure/
│   │   ├── persistence/
│   │   │   ├── __init__.py     # Package exports
│   │   │   ├── models.py       # SQLAlchemy ORM (wallets, escrows, transactions, outbox)
│   │   │   └── repositories.py # Repository implementations
│   │   ├── broker/
│   │   │   ├── __init__.py     # Package exports
│   │   │   └── outbox_publisher.py  # DatabaseEventPublisher + OutboxPublisherWorker
│   │   ├── payment/
│   │   │   └── vnpay.py        # VNPay Adapter (HMAC-SHA512, generate URL, validate IPN)
│   │   └── mappers/
│   │       └── event_mapper.py # Domain Event → Protobuf
│   └── interfaces/
│       ├── grpc/
│       │   └── servicer.py     # gRPC handlers + Domain Error → gRPC Status mapping
│       └── http/
│           └── router.py       # FastAPI router: /topup, /vnpay-ipn, /vnpay-return, /wallet
├── tests/
│   ├── conftest.py             # Set TESTING=1 trước mọi import
│   ├── test_domain.py          # 5 unit tests — Business Invariants
│   ├── test_application.py     # 5 integration tests — Happy paths
│   ├── test_application_sad_paths.py  # 13 integration tests — Failure paths
│   └── test_http_router.py     # 12 integration tests — HTTP endpoints
├── gen/                        # Generated Protobuf Python code (git-ignored)
├── main.py                     # Entrypoint: asyncio.gather(gRPC, HTTP, Outbox, Kafka)
├── pyproject.toml              # Dependencies & pytest config
├── Makefile                    # Shortcuts: run, test, generate-proto
└── .env.example                # Mẫu biến môi trường
```

---

## ⚡ REST API nhanh

| Method | Endpoint | Mô tả |
|---|---|---|
| `POST` | `/api/v1/finance/topup` | Khởi tạo nạp tiền → VNPay URL |
| `GET` | `/api/v1/finance/vnpay-ipn` | Webhook IPN từ VNPay server |
| `GET` | `/api/v1/finance/vnpay-return` | Trang kết quả thanh toán (HTML) |
| `GET` | `/api/v1/finance/wallet?user_id=` | Truy vấn số dư ví |
| `GET` | `/health` | Health check |

Xem chi tiết tại [`docs/02_api.md`](docs/02_api.md).

---

## 🔌 gRPC Interface (Internal)

Dành cho Booking Service gọi nội bộ. Port `50051`.

| Method | Mô tả |
|---|---|
| `FreezeCoin` | Khóa tiền cọc khi Client đặt Booking |
| `TransferToEscrow` | Chuyển tiền vào Escrow khi Companion chấp nhận |
| `ProcessPayout` | Giải phóng Escrow → credit ví Companion |
| `RefundEscrow` | Hoàn tiền Escrow → ví Client |
| `GetWallet` | Truy vấn số dư ví (lazy init) |

---

## 💎 Quy tắc nghiệp vụ cốt lõi

| Quy tắc | Chi tiết |
|---|---|
| **Tỷ giá** | 1 Kano-Coin = 1,000 VNĐ (cố định) |
| **Làm tròn hoa hồng** | `commission = round(amount * rate)` — Python banker's rounding |
| **Khởi tạo ví (Option B)** | Chủ động qua Kafka `UserRegistered` + Lazy fallback tại API |
| **Idempotency IPN** | Kiểm tra `transaction.status != PENDING` trước khi credit |
| **Zero Cross-Service DB** | Các service khác chỉ được gọi qua gRPC hoặc đọc Kafka events |
| **Transactional Outbox** | DB commit + Event insert là một atomic operation |
