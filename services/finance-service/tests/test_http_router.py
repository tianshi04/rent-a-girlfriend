"""
HTTP Router integration tests using FastAPI TestClient.
Covers /topup, /vnpay-ipn, /vnpay-return, and /wallet endpoints.

NOTE: TESTING=1 is set in tests/conftest.py before any internal module is imported.
This ensures bootstrap uses SQLite and skips DB init at import time.
"""

import hmac
import hashlib
import urllib.parse
import pytest
from typing import List
from httpx import AsyncClient, ASGITransport
from fastapi import Depends

from internal.domain.vo import Money
from internal.domain.events import DomainEvent
from internal.application.port import IEventPublisher
from internal.application.command.finance import FinanceCommandService
from internal.infrastructure.persistence.repositories import (
    WalletRepository,
    EscrowRepository,
    TransactionRepository,
)
from internal.infrastructure.payment.vnpay import VNPayAdapter

# Import bootstrap after conftest.py has set TESTING=1.
# Since bootstrap is already in sys.modules, router.py importing it won't cause
# a circular import error.
from internal.bootstrap import app as bootstrap_app, get_db_session, get_finance_cmd


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

HASH_SECRET = "HTTP_TEST_SECRET"
TMN_CODE = "HTTP_TMN"


class MockEventPublisher(IEventPublisher):
    def __init__(self):
        self.events: List[DomainEvent] = []

    def publish(self, event: DomainEvent) -> None:
        self.events.append(event)

    def clear(self):
        self.events = []


def build_signed_ipn(params: dict, secret: str) -> dict:
    """Build VNPay-signed IPN query params."""
    base = {k: v for k, v in params.items() if k != "vnp_SecureHash"}
    sorted_params = sorted(base.items())
    raw_query = "&".join(f"{k}={v}" for k, v in sorted_params)
    secure_hash = (
        hmac.new(
            key=secret.encode("utf-8"),
            msg=raw_query.encode("utf-8"),
            digestmod=hashlib.sha512,
        )
        .hexdigest()
        .upper()
    )
    return {**base, "vnp_SecureHash": secure_hash}


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def mock_publisher():
    return MockEventPublisher()


@pytest.fixture
def vnpay_adapter():
    return VNPayAdapter(
        tmn_code=TMN_CODE,
        hash_secret=HASH_SECRET,
        payment_url="https://sandbox.vnpayment.vn",
        return_url="http://localhost:8080/return",
    )


@pytest.fixture
async def test_app(TestSessionLocal, mock_publisher, vnpay_adapter):
    """
    Reuse the bootstrap_app with dependencies overridden to use
    per-test isolated SQLite in-memory database.
    """

    async def override_get_db():
        async with TestSessionLocal() as db:
            try:
                yield db
                await db.commit()
            except Exception:
                await db.rollback()
                raise

    async def override_get_finance_cmd(db=Depends(override_get_db)):
        return FinanceCommandService(
            wallet_repo=WalletRepository(db),
            escrow_repo=EscrowRepository(db),
            transaction_repo=TransactionRepository(db),
            event_publisher=mock_publisher,
            vnpay_adapter=vnpay_adapter,
        )

    bootstrap_app.dependency_overrides[get_db_session] = override_get_db
    bootstrap_app.dependency_overrides[get_finance_cmd] = override_get_finance_cmd
    yield bootstrap_app
    # Clean up overrides after each test
    bootstrap_app.dependency_overrides.clear()


@pytest.fixture
async def client(test_app):
    async with AsyncClient(
        transport=ASGITransport(app=test_app), base_url="http://test"
    ) as ac:
        yield ac


# ---------------------------------------------------------------------------
# POST /api/v1/finance/topup
# ---------------------------------------------------------------------------


async def test_topup_returns_payment_url(client):
    """POST /topup with valid payload returns 201 and a payment URL."""
    response = await client.post(
        "/api/v1/finance/topup",
        json={"userId": "user-http-1", "amount": 50},
    )
    assert response.status_code == 201
    body = response.json()
    assert "paymentUrl" in body
    assert "vnp_TxnRef" in body["paymentUrl"]


async def test_topup_zero_amount_rejected(client):
    """POST /topup with amount=0 is rejected by Pydantic (gt=0 constraint)."""
    response = await client.post(
        "/api/v1/finance/topup",
        json={"userId": "user-http-2", "amount": 0},
    )
    assert response.status_code == 422  # Unprocessable Entity


async def test_topup_negative_amount_rejected(client):
    """POST /topup with negative amount is rejected by Pydantic."""
    response = await client.post(
        "/api/v1/finance/topup",
        json={"userId": "user-http-3", "amount": -5},
    )
    assert response.status_code == 422


# ---------------------------------------------------------------------------
# GET /api/v1/finance/vnpay-ipn
# ---------------------------------------------------------------------------


async def test_vnpay_ipn_success_flow(
    client, TestSessionLocal, mock_publisher, vnpay_adapter
):
    """GET /vnpay-ipn with valid signature and correct amount credits wallet."""
    # 1. Initiate topup to create PENDING transaction (call service directly)
    async with TestSessionLocal() as db:
        svc = FinanceCommandService(
            wallet_repo=WalletRepository(db),
            escrow_repo=EscrowRepository(db),
            transaction_repo=TransactionRepository(db),
            event_publisher=mock_publisher,
            vnpay_adapter=vnpay_adapter,
        )
        payment_url = await svc.initiate_topup(
            user_id="user-http-ipn-1",
            amount_coins=30,
            client_ip="127.0.0.1",
        )
        await db.commit()

    parsed = urllib.parse.urlparse(payment_url)
    txn_ref = urllib.parse.parse_qs(parsed.query)["vnp_TxnRef"][0]

    # 2. Build signed IPN (30 coins * 1000 VND * 100 cents = 3,000,000)
    ipn_params = build_signed_ipn(
        {
            "vnp_TmnCode": TMN_CODE,
            "vnp_Amount": "3000000",
            "vnp_TxnRef": txn_ref,
            "vnp_ResponseCode": "00",
            "vnp_TransactionNo": "99999",
        },
        HASH_SECRET,
    )

    response = await client.get("/api/v1/finance/vnpay-ipn", params=ipn_params)
    assert response.status_code == 200
    assert response.json()["RspCode"] == "00"


async def test_vnpay_ipn_invalid_signature_returns_97(client):
    """GET /vnpay-ipn with tampered signature returns RspCode 97."""
    params = {
        "vnp_TmnCode": TMN_CODE,
        "vnp_Amount": "5000000",
        "vnp_TxnRef": "tampered-txn",
        "vnp_ResponseCode": "00",
        "vnp_SecureHash": "TAMPERED_HASH",
    }
    response = await client.get("/api/v1/finance/vnpay-ipn", params=params)
    assert response.status_code == 200
    assert response.json()["RspCode"] == "97"


async def test_vnpay_ipn_duplicate_returns_02(
    client, TestSessionLocal, mock_publisher, vnpay_adapter
):
    """GET /vnpay-ipn called twice returns RspCode 02 on second call (idempotency)."""
    async with TestSessionLocal() as db:
        svc = FinanceCommandService(
            wallet_repo=WalletRepository(db),
            escrow_repo=EscrowRepository(db),
            transaction_repo=TransactionRepository(db),
            event_publisher=mock_publisher,
            vnpay_adapter=vnpay_adapter,
        )
        payment_url = await svc.initiate_topup("user-http-ipn-2", 20, "127.0.0.1")
        await db.commit()

    parsed = urllib.parse.urlparse(payment_url)
    txn_ref = urllib.parse.parse_qs(parsed.query)["vnp_TxnRef"][0]

    ipn_params = build_signed_ipn(
        {
            "vnp_TmnCode": TMN_CODE,
            "vnp_Amount": "2000000",
            "vnp_TxnRef": txn_ref,
            "vnp_ResponseCode": "00",
        },
        HASH_SECRET,
    )

    # First call succeeds
    r1 = await client.get("/api/v1/finance/vnpay-ipn", params=ipn_params)
    assert r1.json()["RspCode"] == "00"

    # Second call is idempotency-rejected
    r2 = await client.get("/api/v1/finance/vnpay-ipn", params=ipn_params)
    assert r2.json()["RspCode"] == "02"


# ---------------------------------------------------------------------------
# GET /api/v1/finance/vnpay-return
# ---------------------------------------------------------------------------


async def test_vnpay_return_success_renders_html(client):
    """GET /vnpay-return with valid success params renders HTML success page."""
    params = build_signed_ipn(
        {
            "vnp_TxnRef": "txn-return-1",
            "vnp_ResponseCode": "00",
            "vnp_Amount": "5000000",
        },
        HASH_SECRET,
    )
    response = await client.get("/api/v1/finance/vnpay-return", params=params)
    assert response.status_code == 200
    assert "text/html" in response.headers["content-type"]
    assert "SUCCESS" in response.text


async def test_vnpay_return_failed_renders_failed_html(client):
    """GET /vnpay-return with failed response code renders HTML failure page."""
    params = build_signed_ipn(
        {
            "vnp_TxnRef": "txn-return-2",
            "vnp_ResponseCode": "24",
            "vnp_Amount": "5000000",
        },
        HASH_SECRET,
    )
    response = await client.get("/api/v1/finance/vnpay-return", params=params)
    assert response.status_code == 200
    assert "FAILED" in response.text


async def test_vnpay_return_invalid_sig_renders_failed_html(client):
    """GET /vnpay-return with invalid signature renders HTML failure page."""
    params = {
        "vnp_TxnRef": "txn-return-3",
        "vnp_ResponseCode": "00",
        "vnp_Amount": "5000000",
        "vnp_SecureHash": "INVALID",
    }
    response = await client.get("/api/v1/finance/vnpay-return", params=params)
    assert response.status_code == 200
    assert "FAILED" in response.text


# ---------------------------------------------------------------------------
# GET /api/v1/finance/wallet
# ---------------------------------------------------------------------------


async def test_get_wallet_lazy_creates_wallet(client):
    """GET /wallet for a new user auto-creates a wallet and returns balance = 0."""
    response = await client.get(
        "/api/v1/finance/wallet", params={"userId": "user-http-wallet-1"}
    )
    assert response.status_code == 200
    body = response.json()
    assert body["userId"] == "user-http-wallet-1"
    assert body["availableBalance"] == 0
    assert body["frozenBalance"] == 0
    assert "walletId" in body


async def test_get_wallet_returns_existing_wallet(
    client, TestSessionLocal, mock_publisher, vnpay_adapter
):
    """GET /wallet returns correct balance for existing wallet."""
    user_id = "user-http-wallet-2"

    # Pre-seed wallet with coins via the same test TestSessionLocal
    async with TestSessionLocal() as db:
        svc = FinanceCommandService(
            wallet_repo=WalletRepository(db),
            escrow_repo=EscrowRepository(db),
            transaction_repo=TransactionRepository(db),
            event_publisher=mock_publisher,
            vnpay_adapter=vnpay_adapter,
        )
        wallet = await svc.get_or_create_wallet(user_id)
        wallet.available_balance = wallet.available_balance.add(Money(75))
        await svc.wallet_repo.save(wallet)
        await db.commit()

    response = await client.get("/api/v1/finance/wallet", params={"userId": user_id})
    assert response.status_code == 200
    body = response.json()
    assert body["availableBalance"] == 75
    assert body["userId"] == user_id


async def test_get_wallet_missing_user_id_rejected(client):
    """GET /wallet without user_id query param returns 422."""
    response = await client.get("/api/v1/finance/wallet")
    assert response.status_code == 422
