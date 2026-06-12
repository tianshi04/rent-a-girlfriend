# FINANCE CONTEXT

**Phân loại Subdomain:** Core Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)

Quản lý toàn bộ vòng đời tài chính của nền tảng:
- **Ví Kano-Coin (Wallet):** Nạp tiền, đóng băng/giải phóng số dư
- **Quỹ giữ tiền trung gian (Escrow):** Bảo vệ tiền cọc khi booking đang xử lý
- **Nhật ký giao dịch (Ledger):** Audit log bất biến mọi biến động tài chính
- **Cổng thanh toán VNPay:** Nạp tiền thực qua VNPay Sandbox (HMAC-SHA512 IPN)

## 2. AGGREGATES & ENTITIES

### Aggregate Root 1: `Wallet`
Quản lý số dư của User. Đảm bảo số dư không bao giờ âm.
- **State:** `wallet_id`, `user_id`, `available_balance` (Money), `frozen_balance` (Money)
- **Behaviors:** `topup()`, `freeze_coin()`, `unfreeze_coin()`, `deduct_frozen()`, `deposit()`

### Aggregate Root 2: `Escrow`
Quản lý khoản tiền bị khóa tạm thời cho một Booking cụ thể.
- **State:** `escrow_id`, `booking_id`, `amount` (Money), `status` (HELD → PAID_OUT | REFUNDED)
- **Behaviors:** `payout(companion_id, commission_rate)`, `refund(client_id, refund_amount)`

### Aggregate Root 3: `Transaction`
Audit Log bất biến cho mọi biến động tài chính.
- **State:** `transaction_id`, `user_id`, `amount` (Money), `type`, `status`, `reference_id`
- **Types:** `TOPUP`, `BOOKING_RESERVATION`, `ESCROW_RELEASE`, `REFUND`
- **Statuses:** `PENDING` → `SUCCESS` | `FAILED`

## 3. VALUE OBJECTS

- **`Money`**: Đóng gói số lượng Kano-Coin kiểu số nguyên (`int`). Ném `InvalidAmountError` nếu âm. Có `add()`, `subtract()`, `zero()`.
- **IDs:** `wallet_id`, `escrow_id`, `transaction_id` — UUID string.

> **Tỷ giá cố định:** 1 Kano-Coin = 1,000 VNĐ (xử lý bởi `CurrencyExchangeService`).

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)

| ID | Quy tắc |
|---|---|
| `[INV-F01]` | `available_balance` không bao giờ được `< 0`. |
| `[INV-F02]` | Số tiền yêu cầu `freeze_coin` không được vượt quá `available_balance`. |
| `[INV-F03]` | Khi `unfreeze_coin`, số tiền hoàn trả không được lớn hơn `frozen_balance`. |
| `[INV-F04]` | Mỗi `booking_id` chỉ được có duy nhất 1 `Escrow` ở trạng thái `HELD`. |
| `[INV-F05]` | Chỉ được `payout` hoặc `refund` khi `Escrow.status == HELD`. |

## 5. DOMAIN EVENTS

| Sự kiện | Phát ra khi | Payload chính |
|---|---|---|
| `WalletToppedUp` | VNPay IPN xác nhận thành công | `wallet_id`, `user_id`, `amount`, `vnpay_amount_vnd` |
| `CoinsFrozen` | Client đặt cọc booking | `wallet_id`, `user_id`, `amount`, `booking_id` |
| `CoinsUnfrozen` | Hủy đặt cọc trước khi escrow | `wallet_id`, `user_id`, `amount`, `booking_id` |
| `EscrowCreated` | Companion chấp nhận booking | `booking_id`, `amount` |
| `PayoutProcessed` | Booking hoàn thành, giải phóng escrow | `booking_id`, `companion_id`, `amount`, `commission_amount`, `net_amount` |
| `EscrowRefunded` | Booking bị hủy, hoàn tiền Client | `booking_id`, `client_id`, `refund_amount` |

> Tất cả sự kiện được xuất bản qua **Transactional Outbox** → Kafka topic `finance.events` theo định dạng **CloudEvents JSON**.

## 6. COMMANDS & USE CASES

| Command | Input | Kết quả |
|---|---|---|
| `FreezeCoin` | `user_id`, `amount`, `booking_id` | Đóng băng số dư, tạo Transaction `BOOKING_RESERVATION/PENDING` |
| `TransferToEscrow` | `user_id`, `amount`, `booking_id` | Chuyển frozen → Escrow `HELD`, Transaction → `SUCCESS` |
| `ProcessPayout` | `booking_id`, `companion_id`, `commission_rate` | Giải phóng Escrow, credit ví Companion (net sau hoa hồng) |
| `RefundEscrow` | `booking_id`, `client_id`, `refund_amount` | Hoàn tiền Escrow về ví Client |
| `InitiateTopup` | `user_id`, `amount_coins`, `client_ip` | Tạo Transaction `TOPUP/PENDING`, trả URL thanh toán VNPay |
| `ProcessVNPayIPN` | `vnp_*` params | Xác thực HMAC, credit ví, idempotency guard |

## 7. DOMAIN SERVICES

- **`CommissionCalculatorService`**: Tính hoa hồng nền tảng. `commission = round(amount * rate)` (Python banker's rounding).
- **`CurrencyExchangeService`**: Quy đổi VNĐ ↔ Kano-Coin. Tỷ giá `RATE = 1,000`.

## 8. REPOSITORIES (INTERFACES)

| Interface | Methods |
|---|---|
| `IWalletRepository` | `save()`, `find_by_id()`, `find_by_user_id()` |
| `IEscrowRepository` | `save()`, `find_by_id()`, `find_by_booking_id()` |
| `ITransactionRepository` | `save()`, `find_by_id()`, `find_by_reference_id(ref_id, type)` |

## 9. INTEGRATION

### Kafka Topics
| Topic | Vai trò |
|---|---|
| `finance.events` | **Publish** — Finance Service xuất bản tất cả domain events |
| `identity.events` | **Subscribe** — Finance Service lắng nghe `UserRegistered` để onboard ví |

### gRPC Interface (Internal — được gọi bởi Booking Service)
| Method | Mô tả |
|---|---|
| `FreezeCoin` | Khóa tiền khi Client yêu cầu Booking |
| `TransferToEscrow` | Chuyển tiền vào Escrow khi Companion chấp nhận |
| `ProcessPayout` | Giải phóng Escrow sau khi Booking hoàn thành |
| `RefundEscrow` | Hoàn tiền khi Booking bị hủy |
| `GetWallet` | Truy vấn số dư ví (với lazy init fallback) |

### REST Interface (External — Client Mobile App)
| Method | Endpoint | Mô tả |
|---|---|---|
| POST | `/api/v1/finance/topup` | Khởi tạo lệnh nạp tiền, trả VNPay URL |
| GET | `/api/v1/finance/vnpay-ipn` | Webhook IPN từ VNPay (HMAC-SHA512) |
| GET | `/api/v1/finance/vnpay-return` | Trang kết quả thanh toán (HTML) |
| GET | `/api/v1/finance/wallet` | Truy vấn số dư ví theo `user_id` |
