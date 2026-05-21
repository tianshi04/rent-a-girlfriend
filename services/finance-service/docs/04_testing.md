# Testing Guide — Finance Service

## Chiến lược kiểm thử (Test Strategy)

Finance Service áp dụng 3 tầng kiểm thử với mục tiêu rõ ràng:

| Tầng | File | Mục tiêu | DB |
|---|---|---|---|
| **Unit — Domain** | `test_domain.py` | Kiểm chứng tất cả Business Invariants, logic nghiệp vụ thuần túy | Không cần |
| **Integration — Application** | `test_application.py`, `test_application_sad_paths.py` | Kiểm chứng Use Cases đầu-cuối với DB giả lập | SQLite in-memory |
| **Integration — HTTP** | `test_http_router.py` | Kiểm chứng REST endpoints, middleware DI, VNPay IPN flow | SQLite in-memory |

**Tổng số test cases: 35**

---

## Chạy Test Suite

```bash
make test
# Tương đương: PYTHONPATH=$PWD/gen uv run pytest -v
```

**Kết quả kỳ vọng:**
```
35 passed in ~7s
```

---

## Chi tiết từng test file

### `tests/test_domain.py` — 5 tests (Pure Unit)

Không cần DB, không cần mock. Kiểm thử trực tiếp các domain object.

| Test | Bao phủ |
|---|---|
| `test_money_validation` | `Money` VO: add/subtract/negative/zero |
| `test_wallet_invariants_and_events` | [INV-F01] [INV-F02] [INV-F03] + domain events |
| `test_escrow_invariants_and_commission` | [INV-F04] [INV-F05] + commission rounding |
| `test_escrow_refund` | Refund từ HELD status |
| `test_domain_services` | `CurrencyExchangeService`, `CommissionCalculatorService` |

---

### `tests/test_application.py` — 5 tests (Integration — Happy Paths)

Sử dụng SQLite in-memory + Mock Event Publisher. Kiểm thử toàn bộ luồng nghiệp vụ.

| Test | Luồng |
|---|---|
| `test_wallet_lazy_onboarding` | Lazy-create ví khi truy vấn lần đầu |
| `test_wallet_active_onboarding_idempotency` | Kafka-driven onboarding không tạo ví trùng |
| `test_booking_reservation_flow` | Freeze → TransferToEscrow → Payout (full cycle) |
| `test_escrow_refund_flow` | Freeze → TransferToEscrow → RefundEscrow |
| `test_vnpay_topup_and_ipn_flow` | InitiateTopup → IPN success → Idempotency check |

---

### `tests/test_application_sad_paths.py` — 13 tests (Integration — Failure Paths)

| Test | Lỗi dự kiến |
|---|---|
| `test_freeze_coin_insufficient_balance` | `InsufficientBalanceError` [INV-F02] |
| `test_freeze_coin_negative_amount` | `InvalidAmountError` [INV-F01] |
| `test_transfer_to_escrow_wallet_not_found` | `WalletNotFoundError` |
| `test_transfer_to_escrow_duplicate_escrow` | `EscrowAlreadyExistsError` [INV-F04] |
| `test_process_payout_escrow_not_found` | `EscrowNotFoundError` |
| `test_process_payout_already_paid_out` | `InvalidEscrowStatusTransitionError` [INV-F05] |
| `test_refund_escrow_not_found` | `EscrowNotFoundError` |
| `test_refund_escrow_already_paid_out` | `InvalidEscrowStatusTransitionError` [INV-F05] |
| `test_refund_escrow_already_refunded` | `InvalidEscrowStatusTransitionError` [INV-F05] |
| `test_vnpay_ipn_invalid_signature` | `RspCode: "97"` |
| `test_vnpay_ipn_order_not_found` | `RspCode: "01"` |
| `test_vnpay_ipn_amount_mismatch` | `RspCode: "04"` |
| `test_vnpay_ipn_failed_payment_marks_transaction_failed` | `txn.status == "FAILED"`, wallet không bị credit |

---

### `tests/test_http_router.py` — 12 tests (Integration — HTTP Layer)

Sử dụng `httpx.AsyncClient` + FastAPI `dependency_overrides`. Kiểm thử từ HTTP request đến DB.

| Test | Endpoint | Kiểm thử |
|---|---|---|
| `test_topup_returns_payment_url` | `POST /topup` | 201 + có `vnp_TxnRef` trong URL |
| `test_topup_zero_amount_rejected` | `POST /topup` | 422 (Pydantic validation) |
| `test_topup_negative_amount_rejected` | `POST /topup` | 422 (Pydantic validation) |
| `test_vnpay_ipn_success_flow` | `GET /vnpay-ipn` | `RspCode: "00"`, ví được credit |
| `test_vnpay_ipn_invalid_signature_returns_97` | `GET /vnpay-ipn` | `RspCode: "97"` |
| `test_vnpay_ipn_duplicate_returns_02` | `GET /vnpay-ipn` | `RspCode: "02"` (idempotency) |
| `test_vnpay_return_success_renders_html` | `GET /vnpay-return` | HTML với "SUCCESS" |
| `test_vnpay_return_failed_renders_failed_html` | `GET /vnpay-return` | HTML với "FAILED" |
| `test_vnpay_return_invalid_sig_renders_failed_html` | `GET /vnpay-return` | HTML với "FAILED" |
| `test_get_wallet_lazy_creates_wallet` | `GET /wallet` | 200, balance = 0 |
| `test_get_wallet_returns_existing_wallet` | `GET /wallet` | 200, balance = 75 |
| `test_get_wallet_missing_user_id_rejected` | `GET /wallet` | 422 |

---

## Cấu hình Test (`pyproject.toml`)

```toml
[tool.pytest.ini_options]
asyncio_mode = "auto"
asyncio_default_fixture_loop_scope = "function"
```

- `asyncio_mode = "auto"`: Tất cả `async def test_*` tự động được pytest-asyncio chạy
- Mỗi test function có event loop riêng biệt (`function` scope)

---

## Isolation Strategy

- **Domain tests:** Không có side effects — không cần setup
- **Application tests:** Mỗi test dùng fixture `test_db_session` tạo engine SQLite in-memory riêng → tự động teardown sau test
- **HTTP tests:** Mỗi test dùng fixture `db_engine` (function scope) → `bootstrap_app.dependency_overrides` bị clear sau mỗi test

---

## Thêm test mới

1. **Domain test:** Import aggregate/VO trực tiếp, không cần fixture
2. **Application test:** Dùng fixture `finance_service` đã có sẵn trong `test_application.py` / `test_application_sad_paths.py`
3. **HTTP test:** Dùng fixture `client` trong `test_http_router.py`

> **Lưu ý:** `tests/conftest.py` set `os.environ["TESTING"] = "1"` trước mọi import để bootstrap dùng SQLite và bỏ qua `asyncio.run(init_db())`.
