# API Reference — Finance Service

Finance Service cung cấp hai loại interface: **REST API** (cho Client Mobile App) và **gRPC** (cho internal service-to-service communication).

---

## REST API — FastAPI (Port 8080)

Base URL: `http://<host>:8080/api/v1/finance`

### `POST /topup`

Khởi tạo lệnh nạp tiền Kano-Coin qua VNPay. Tạo Transaction `PENDING` và trả về URL thanh toán.

**Request Body:**
```json
{
  "user_id": "uuid-of-user",
  "amount": 50
}
```

> `amount` là số Kano-Coin muốn nạp (phải `> 0`). 50 Coin = 50,000 VNĐ.

**Response `201 Created`:**
```json
{
  "payment_url": "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_TxnRef=...&vnp_SecureHash=..."
}
```

**Lỗi:**
- `422 Unprocessable Entity`: `amount <= 0`
- `500 Internal Server Error`: DB lỗi

---

### `GET /vnpay-ipn`

**VNPay IPN Webhook.** VNPay server gọi endpoint này sau khi người dùng thanh toán. Xử lý bất đồng bộ, credit ví, cập nhật Transaction.

> **Không gọi endpoint này từ frontend.** Chỉ dành cho VNPay server.

**Query params:** Các tham số `vnp_*` chuẩn VNPay (bao gồm `vnp_SecureHash`).

**Response `200 OK`:**
```json
{ "RspCode": "00", "Message": "Confirm success" }
```

| `RspCode` | Ý nghĩa |
|---|---|
| `00` | Xử lý thành công |
| `97` | Chữ ký HMAC-SHA512 không hợp lệ |
| `01` | `vnp_TxnRef` không tìm thấy trong DB |
| `02` | Giao dịch đã được xử lý (idempotency guard) |
| `04` | Số tiền không khớp với Transaction cục bộ |
| `99` | Internal server error |

---

### `GET /vnpay-return`

Trang kết quả thanh toán. Người dùng được redirect tới đây sau khi hoàn tất luồng VNPay.

**Query params:** Các tham số `vnp_*` chuẩn VNPay.

**Response `200 OK`:** HTML page (glassmorphism UI hiển thị kết quả SUCCESS / FAILED).

---

### `GET /wallet`

Truy vấn số dư ví theo `user_id`. Nếu ví chưa tồn tại, **lazy-create** ví mới với balance = 0.

**Query params:**
- `user_id` *(required)*: UUID của user cần truy vấn

**Response `200 OK`:**
```json
{
  "wallet_id": "uuid",
  "user_id": "uuid",
  "available_balance": 100,
  "frozen_balance": 20
}
```

**Lỗi:**
- `422 Unprocessable Entity`: thiếu `user_id`
- `500 Internal Server Error`: DB lỗi

---

### `GET /health`

**Response `200 OK`:**
```json
{ "status": "ok", "service": "finance-service" }
```

---

## gRPC — (Port 50051)

**Proto contract:** [`contracts/finance/v1/service/finance_service.proto`](../../../contracts/finance/v1/service/finance_service.proto)

> Chỉ dành cho **internal service-to-service** (Booking Service gọi Finance Service). Không expose ra ngoài internet. Được bảo vệ bởi Istio mTLS (SPIFFE identity).

### `FreezeCoin`

Khóa tiền cọc trong ví Client khi Client yêu cầu Booking.

```protobuf
rpc FreezeCoin(FreezeCoinRequest) returns (FinanceCommandResponse);

message FreezeCoinRequest {
  string user_id    = 1;
  int32  amount     = 2;
  string booking_id = 3;
}
```

**gRPC Status Codes:**
- `OK` — Thành công, trả `transaction_id`
- `NOT_FOUND` — Ví không tồn tại
- `FAILED_PRECONDITION` — Số dư không đủ hoặc `amount <= 0`

---

### `TransferToEscrow`

Chuyển tiền từ frozen balance của Client vào Escrow khi Companion chấp nhận Booking.

```protobuf
rpc TransferToEscrow(TransferToEscrowRequest) returns (FinanceCommandResponse);

message TransferToEscrowRequest {
  string user_id    = 1;
  int32  amount     = 2;
  string booking_id = 3;
}
```

**gRPC Status Codes:**
- `OK` — Thành công, trả `escrow_id`
- `NOT_FOUND` — Ví không tồn tại
- `ALREADY_EXISTS` — [INV-F04] Escrow cho booking này đã ở trạng thái HELD
- `FAILED_PRECONDITION` — Frozen balance không đủ

---

### `ProcessPayout`

Giải phóng Escrow sau khi Booking hoàn thành. Deduct hoa hồng hệ thống, credit phần còn lại vào ví Companion.

```protobuf
rpc ProcessPayout(ProcessPayoutRequest) returns (FinanceCommandResponse);

message ProcessPayoutRequest {
  string booking_id      = 1;
  string companion_id    = 2;
  float  commission_rate = 3;  // Ví dụ: 0.1 = 10%
}
```

**gRPC Status Codes:**
- `OK` — Thành công, trả `transaction_id`
- `NOT_FOUND` — Escrow không tồn tại
- `FAILED_PRECONDITION` — [INV-F05] Escrow không ở trạng thái HELD

---

### `RefundEscrow`

Hoàn tiền từ Escrow về ví Client khi Booking bị hủy.

```protobuf
rpc RefundEscrow(RefundEscrowRequest) returns (FinanceCommandResponse);

message RefundEscrowRequest {
  string booking_id    = 1;
  string client_id     = 2;
  int32  refund_amount = 3;
}
```

**gRPC Status Codes:**
- `OK` — Thành công, trả `transaction_id`
- `NOT_FOUND` — Escrow không tồn tại
- `FAILED_PRECONDITION` — [INV-F05] Escrow không ở trạng thái HELD

---

### `GetWallet`

Truy vấn số dư ví. Nếu chưa tồn tại, lazy-create ví mới.

```protobuf
rpc GetWallet(GetWalletRequest) returns (GetWalletResponse);

message GetWalletRequest {
  string user_id = 1;
}

message GetWalletResponse {
  string wallet_id          = 1;
  string user_id            = 2;
  int32  available_balance  = 3;
  int32  frozen_balance     = 4;
}
```

---

### `CheckBalance`

Kiểm tra số dư khả dụng của ví Client trước khi booking. Nếu chưa có ví, tự động khởi tạo (lazy-create) ví mới.

```protobuf
rpc CheckBalance(CheckBalanceRequest) returns (CheckBalanceResponse);

message CheckBalanceRequest {
  string user_id = 1;
  int64  amount  = 2;
}

message CheckBalanceResponse {
  bool has_sufficient_balance = 1;
}
```

**gRPC Status Codes:**
- `OK` — Thành công, trả về trạng thái số dư đủ/thiếu qua field `has_sufficient_balance`
- `FAILED_PRECONDITION` — `amount` truyền vào không hợp lệ (`amount < 0`)

---

## Response chung cho Commands gRPC

```protobuf
message FinanceCommandResponse {
  string transaction_id = 1;  // ID của transaction hoặc escrow vừa tạo
  string status         = 2;  // "SUCCESS"
  string message        = 3;  // Mô tả kết quả
}
```
