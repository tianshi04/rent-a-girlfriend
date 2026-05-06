# FINANCE CONTEXT

**Phân loại Subdomain:** Core Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Quản lý ví ảo (Kano-Coin), xử lý giao dịch, giữ tiền cọc (Escrow) và tính toán hoa hồng.

## 2. AGGREGATES & ENTITIES
### Aggregate Root 1: `Wallet`
Quản lý số dư của User, đảm bảo không bao giờ âm tiền.
*   **State:** `WalletId`, `UserId`, `AvailableBalance`, `FrozenBalance`.
*   **Entities con:** `TransactionLine` (Lưu vết thay đổi số dư trực tiếp trong Aggregate Wallet để đảm bảo tính nhất quán nội bộ).

### Aggregate Root 2: `Transaction`
Đóng vai trò Audit Log cho mọi biến động tài chính (Nạp, Rút, Thanh toán).
*   **State:** `TransactionId`, `UserId`, `Amount`, `Type`, `Status`, `ReferenceId`.

### Aggregate Root 3: `Escrow`
Quản lý khoản tiền bị khóa tạm thời (quỹ đảm bảo) cho một Booking cụ thể.
*   **State:** `EscrowId`, `BookingId`, `Amount`, `Status` (HELD, REFUNDED, PAID_OUT).

## 3. VALUE OBJECTS
*   `Money`: Đóng gói giá trị số tiền (Amount) và đơn vị tiền tệ (Currency - mặc định là Kano-Coin). Có các hàm cộng, trừ, và ném lỗi nếu số dư âm.
*   `WalletId`, `EscrowId`, `TransactionId`: Định danh (UUID).

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)
*   `[INV-F01]` `AvailableBalance` không bao giờ được `< 0`.
*   `[INV-F02]` Số tiền yêu cầu Freeze không được vượt quá `AvailableBalance`.
*   `[INV-F03]` Khi Unfreeze, số tiền trả về không được lớn hơn `FrozenBalance`.
*   `[INV-F04]` Mỗi `BookingId` chỉ được có duy nhất 1 `Escrow` ở trạng thái `HELD`.
*   `[INV-F05]` Chỉ được Payout hoặc Refund khi Status đang là `HELD`.

## 5. THIẾT KẾ COMMAND & EVENT
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `FreezeCoin` | `walletId`, `amount`, `referenceId` (bookingId) | `CoinFrozen` { walletId, amount, refId } |
| `TransferToEscrow` | `walletId`, `bookingId`, `amount` | `CoinEscrowed` { bookingId, amount } |
| `ProcessPayout` | `escrowId`, `companionWalletId`, `commissionRate` | `PayoutProcessed` { bookingId, amount, commission } |

## 6. DOMAIN SERVICES
*   `CommissionCalculatorService`: Chứa thuật toán tính toán hoa hồng của nền tảng khi một Booking hoàn thành (ví dụ: nền tảng thu 10% - 20% tùy thuộc vào hạng của Companion).
*   `CurrencyExchangeService`: Chuyển đổi từ tiền VNĐ thực tế sang Kano-Coin khi Nạp tiền (Deposit) thông qua cổng thanh toán VNPay.

## 7. REPOSITORIES
*   `IWalletRepository`: Lưu trữ trạng thái Wallet.
*   `IEscrowRepository`: Lưu trữ trạng thái Escrow.
