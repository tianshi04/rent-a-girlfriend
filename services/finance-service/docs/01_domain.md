# Domain Model — Finance Service

Tài liệu mô tả chi tiết thiết kế Domain Layer của Finance Service theo chuẩn DDD.

## Value Object: `Money`

**File:** [`internal/domain/vo.py`](../internal/domain/vo.py)

```python
Money(amount: int)  # Kano-Coin, chỉ chấp nhận >= 0
Money.zero()        # Tạo Money(0)
m1.add(m2)          # Trả Money mới
m1.subtract(m2)     # Ném InvalidAmountError nếu kết quả < 0
```

**Quy tắc:** `Money` là immutable (frozen dataclass). Mọi phép tính luôn sinh ra đối tượng mới — không có side effect.

---

## Aggregate: `Wallet`

**File:** [`internal/domain/aggregate/wallet.py`](../internal/domain/aggregate/wallet.py)

| State | Kiểu | Mô tả |
|---|---|---|
| `wallet_id` | `str` (UUID) | Định danh ví |
| `user_id` | `str` (UUID) | Chủ sở hữu |
| `available_balance` | `Money` | Số dư có thể dùng |
| `frozen_balance` | `Money` | Số dư đang bị đóng băng |

### Behaviors & Invariants

| Method | Invariant | Event phát ra |
|---|---|---|
| `Wallet.create(wallet_id, user_id)` | Khởi tạo với balance = 0 | — |
| `topup(amount, transaction_id, vnpay_amount_vnd)` | [INV-F01] | `WalletToppedUp` |
| `freeze_coin(amount, booking_id)` | [INV-F01] [INV-F02] available -= amount, frozen += amount | `CoinsFrozen` |
| `unfreeze_coin(amount, booking_id)` | [INV-F03] frozen -= amount, available += amount | `CoinsUnfrozen` |
| `deduct_frozen(amount)` | [INV-F03] frozen -= amount (chuyển sang Escrow) | — |
| `deposit(amount)` | [INV-F01] available += amount (nhận tiền từ Escrow/Payout) | — |

---

## Aggregate: `Escrow`

**File:** [`internal/domain/aggregate/escrow.py`](../internal/domain/aggregate/escrow.py)

| State | Kiểu | Mô tả |
|---|---|---|
| `escrow_id` | `str` (UUID) | Định danh |
| `booking_id` | `str` (UUID) | Booking liên kết |
| `amount` | `Money` | Số tiền đang giữ |
| `status` | `str` | `HELD` → `PAID_OUT` hoặc `REFUNDED` |

### Behaviors & Invariants

| Method | Guard | Event phát ra |
|---|---|---|
| `Escrow.create(escrow_id, booking_id, amount)` | — | `EscrowCreated` |
| `payout(companion_id, commission_rate)` | [INV-F05] status == HELD | `PayoutProcessed` |
| `refund(client_id, refund_amount)` | [INV-F05] status == HELD | `EscrowRefunded` |

**Tính hoa hồng:**
```
commission = round(amount * commission_rate)   # Python banker's rounding
net_amount  = amount - commission
```

---

## Aggregate: `Transaction`

**File:** [`internal/domain/aggregate/transaction.py`](../internal/domain/aggregate/transaction.py)

Đóng vai trò **Immutable Audit Ledger**. Mỗi biến động tài chính tạo ra một bản ghi Transaction độc lập.

| `type` | Mô tả |
|---|---|
| `TOPUP` | Nạp tiền qua VNPay |
| `BOOKING_RESERVATION` | Đóng băng tiền cọc |
| `ESCROW_RELEASE` | Giải phóng escrow → Companion |
| `REFUND` | Hoàn tiền từ escrow → Client |

| `status` | Mô tả |
|---|---|
| `PENDING` | Giao dịch đang xử lý |
| `SUCCESS` | Hoàn thành |
| `FAILED` | Thất bại (VNPay từ chối) |

---

## Domain Events

**File:** [`internal/domain/events.py`](../internal/domain/events.py)

Tất cả events là **immutable frozen dataclass**, không chứa logic nghiệp vụ. Được phát ra từ Aggregate và lưu vào Outbox để publish lên Kafka.

| Event | Phát ra từ | Payload |
|---|---|---|
| `WalletToppedUp` | `Wallet.topup()` | `wallet_id, user_id, amount, vnpay_amount_vnd` |
| `CoinsFrozen` | `Wallet.freeze_coin()` | `wallet_id, user_id, amount, booking_id` |
| `CoinsUnfrozen` | `Wallet.unfreeze_coin()` | `wallet_id, user_id, amount, booking_id` |
| `EscrowCreated` | `Escrow.create()` | `booking_id, amount` |
| `PayoutProcessed` | `Escrow.payout()` | `booking_id, companion_id, amount, commission_amount, net_amount` |
| `EscrowRefunded` | `Escrow.refund()` | `booking_id, client_id, refund_amount` |
| `CoinsFreezeFailed ` | Đóng băng tiền cọc thất bại | `booking_id`, `client_id`, `reason` |
| `EscrowFailed` | Ký quỹ tiền cọc thất bại | `booking_id`, `client_id`, `reason` |
| `RefundFailed` | Hoàn tiền Escrow thất bại | `booking_id`, `client_id`, `reason` |

---

## Domain Services

**File:** [`internal/domain/service.py`](../internal/domain/service.py)

### `CurrencyExchangeService`
```python
# 1 Kano-Coin = 1,000 VNĐ
CurrencyExchangeService.coin_to_vnd(money: Money) -> int     # amount * 1000
CurrencyExchangeService.vnd_to_coin(vnd: int) -> Money       # vnd // 1000
```

### `CommissionCalculatorService`
```python
CommissionCalculatorService.calculate_commission(amount: Money, rate: float) -> int
# commission = round(amount.amount * rate)   ← Python banker's rounding
```

---

## Domain Errors

**File:** [`internal/domain/errors.py`](../internal/domain/errors.py)

| Class | Khi nào |
|---|---|
| `InvalidAmountError` | `Money` nhận giá trị âm |
| `InsufficientBalanceError` | Freeze > available_balance |
| `InsufficientFrozenBalanceError` | Unfreeze/deduct > frozen_balance |
| `WalletNotFoundError` | Truy vấn ví không tồn tại |
| `WalletAlreadyExistsError` | Onboard ví đã có (nội bộ) |
| `EscrowNotFoundError` | Truy vấn escrow không tồn tại |
| `EscrowAlreadyExistsError` | [INV-F04] Tạo escrow thứ 2 cho cùng booking |
| `InvalidEscrowStatusTransitionError` | [INV-F05] Payout/Refund không từ HELD |
