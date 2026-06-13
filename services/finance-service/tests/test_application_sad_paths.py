"""
Sad-path (failure scenario) tests for FinanceCommandService.
Covers error propagation for all [INV-F01..F05] violation cases at the Application layer.
"""

import pytest
import grpc
from typing import List
from sqlalchemy import select
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

from internal.domain.vo import Money
from internal.domain.events import DomainEvent
from internal.interfaces.grpc.servicer import FinanceServiceServicer
from internal.infrastructure.persistence.models import OutboxModel, TransactionModel
from internal.domain.errors import (
    WalletNotFoundError,
    EscrowNotFoundError,
    InsufficientBalanceError,
    InvalidEscrowStatusTransitionError,
    InvalidAmountError,
)
from internal.application.port import IEventPublisher
from internal.application.command.finance import FinanceCommandService
from internal.infrastructure.persistence.models import Base
from internal.infrastructure.persistence.repositories import (
    WalletRepository,
    EscrowRepository,
    TransactionRepository,
)
from internal.infrastructure.payment.vnpay import VNPayAdapter


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


class MockEventPublisher(IEventPublisher):
    def __init__(self):
        self.events: List[DomainEvent] = []

    def publish(self, event: DomainEvent) -> None:
        self.events.append(event)

    def clear(self):
        self.events = []


@pytest.fixture
async def test_db_session():
    engine = create_async_engine("sqlite+aiosqlite:///:memory:", echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    SessionLocal = async_sessionmaker(
        autocommit=False, autoflush=False, bind=engine, class_=AsyncSession
    )
    async with SessionLocal() as session:
        yield session
    await engine.dispose()


@pytest.fixture
def mock_publisher():
    return MockEventPublisher()


@pytest.fixture
def finance_service(test_db_session, mock_publisher):
    vnpay = VNPayAdapter(
        tmn_code="TMN_TEST",
        hash_secret="SECRET_TEST",
        payment_url="https://sandbox.vnpayment.vn",
        return_url="http://localhost:8080/return",
    )
    return FinanceCommandService(
        wallet_repo=WalletRepository(test_db_session),
        escrow_repo=EscrowRepository(test_db_session),
        transaction_repo=TransactionRepository(test_db_session),
        event_publisher=mock_publisher,
        vnpay_adapter=vnpay,
    )


# ---------------------------------------------------------------------------
# 1. freeze_coin sad paths
# ---------------------------------------------------------------------------


async def test_freeze_coin_insufficient_balance(finance_service):
    """[INV-F02] freeze_coin raises InsufficientBalanceError when coins > available balance."""
    user_id = "u-sad-1"
    # Wallet auto-created with balance = 0 coins
    with pytest.raises(InsufficientBalanceError):
        await finance_service.freeze_coin(
            user_id=user_id,
            amount=50,
            booking_id="b-sad-1",
        )


async def test_freeze_coin_negative_amount(finance_service):
    """[INV-F01] freeze_coin raises InvalidAmountError when amount < 0."""
    user_id = "u-sad-2"
    with pytest.raises(InvalidAmountError):
        await finance_service.freeze_coin(
            user_id=user_id,
            amount=-10,
            booking_id="b-sad-2",
        )


# ---------------------------------------------------------------------------
# 2. transfer_to_escrow sad paths
# ---------------------------------------------------------------------------


async def test_transfer_to_escrow_wallet_not_found(finance_service):
    """transfer_to_escrow raises WalletNotFoundError for non-existent user."""
    with pytest.raises(WalletNotFoundError):
        await finance_service.transfer_to_escrow(
            user_id="ghost-user",
            amount=50,
            booking_id="b-ghost",
        )


async def test_transfer_to_escrow_duplicate_escrow(finance_service, mock_publisher):
    """[INV-F04] transfer_to_escrow raises EscrowAlreadyExistsError on duplicate."""
    client_id = "u-sad-4"
    booking_id = "b-sad-4"

    # Seed wallet with coins
    wallet = await finance_service.get_or_create_wallet(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(200))
    await finance_service.wallet_repo.save(wallet)

    # First freeze + escrow
    await finance_service.freeze_coin(client_id, 100, booking_id)
    await finance_service.transfer_to_escrow(client_id, 100, booking_id)
    mock_publisher.clear()

    # Re-freeze for second attempt
    wallet = await finance_service.wallet_repo.find_by_user_id(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(wallet)
    await finance_service.freeze_coin(client_id, 100, booking_id)

    # Second transfer_to_escrow must raise [INV-F04]
    from internal.domain.errors import EscrowAlreadyExistsError

    with pytest.raises(EscrowAlreadyExistsError):
        await finance_service.transfer_to_escrow(client_id, 100, booking_id)


# ---------------------------------------------------------------------------
# 3. process_payout sad paths
# ---------------------------------------------------------------------------


async def test_process_payout_escrow_not_found(finance_service):
    """process_payout raises EscrowNotFoundError for non-existent booking."""
    with pytest.raises(EscrowNotFoundError):
        await finance_service.process_payout(
            booking_id="ghost-booking",
            companion_id="comp-ghost",
            commission_rate=0.1,
        )


async def test_process_payout_already_paid_out(finance_service, mock_publisher):
    """[INV-F05] process_payout raises InvalidEscrowStatusTransitionError on double payout."""
    client_id = "u-sad-5"
    companion_id = "comp-sad-5"
    booking_id = "b-sad-5"

    wallet = await finance_service.get_or_create_wallet(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(wallet)

    await finance_service.freeze_coin(client_id, 100, booking_id)
    await finance_service.transfer_to_escrow(client_id, 100, booking_id)
    # First payout
    await finance_service.process_payout(booking_id, companion_id, commission_rate=0.1)
    mock_publisher.clear()

    # Second payout must raise [INV-F05]
    with pytest.raises(InvalidEscrowStatusTransitionError):
        await finance_service.process_payout(
            booking_id, companion_id, commission_rate=0.1
        )


# ---------------------------------------------------------------------------
# 4. refund_escrow sad paths
# ---------------------------------------------------------------------------


async def test_refund_escrow_not_found(finance_service):
    """refund_escrow raises EscrowNotFoundError for non-existent booking."""
    with pytest.raises(EscrowNotFoundError):
        await finance_service.refund_escrow(
            booking_id="ghost-booking-2",
            client_id="client-ghost",
            refund_amount=50,
        )


async def test_refund_escrow_already_paid_out(finance_service, mock_publisher):
    """[INV-F05] refund_escrow raises InvalidEscrowStatusTransitionError when escrow is PAID_OUT."""
    client_id = "u-sad-6"
    companion_id = "comp-sad-6"
    booking_id = "b-sad-6"

    wallet = await finance_service.get_or_create_wallet(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(wallet)

    await finance_service.freeze_coin(client_id, 100, booking_id)
    await finance_service.transfer_to_escrow(client_id, 100, booking_id)
    await finance_service.process_payout(booking_id, companion_id, commission_rate=0.1)
    mock_publisher.clear()

    # Refund from PAID_OUT must raise [INV-F05]
    with pytest.raises(InvalidEscrowStatusTransitionError):
        await finance_service.refund_escrow(
            booking_id=booking_id,
            client_id=client_id,
            refund_amount=100,
        )


async def test_refund_escrow_already_refunded(finance_service, mock_publisher):
    """[INV-F05] refund_escrow raises InvalidEscrowStatusTransitionError on double refund."""
    client_id = "u-sad-7"
    booking_id = "b-sad-7"

    wallet = await finance_service.get_or_create_wallet(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(wallet)

    await finance_service.freeze_coin(client_id, 100, booking_id)
    await finance_service.transfer_to_escrow(client_id, 100, booking_id)
    await finance_service.refund_escrow(booking_id, client_id, 100)
    mock_publisher.clear()

    # Second refund must raise [INV-F05]
    with pytest.raises(InvalidEscrowStatusTransitionError):
        await finance_service.refund_escrow(booking_id, client_id, 100)


# ---------------------------------------------------------------------------
# 5. VNPay IPN sad paths
# ---------------------------------------------------------------------------


async def test_vnpay_ipn_invalid_signature(finance_service):
    """IPN with invalid HMAC signature returns RspCode 97."""
    params = {
        "vnp_TmnCode": "TMN_TEST",
        "vnp_Amount": "5000000",
        "vnp_TxnRef": "some-txn-id",
        "vnp_ResponseCode": "00",
        "vnp_SecureHash": "INVALID_HASH",
    }
    response = await finance_service.process_vnpay_ipn(params)
    assert response["RspCode"] == "97"


async def test_vnpay_ipn_order_not_found(finance_service):
    """IPN with unknown txn_ref returns RspCode 01."""
    import hmac
    import hashlib

    params = {
        "vnp_TmnCode": "TMN_TEST",
        "vnp_Amount": "5000000",
        "vnp_TxnRef": "non-existent-txn",
        "vnp_ResponseCode": "00",
    }
    sorted_params = sorted(params.items())
    raw_query = "&".join(f"{k}={v}" for k, v in sorted_params)
    secure_hash = (
        hmac.new(
            key=b"SECRET_TEST",
            msg=raw_query.encode("utf-8"),
            digestmod=hashlib.sha512,
        )
        .hexdigest()
        .upper()
    )
    params["vnp_SecureHash"] = secure_hash

    response = await finance_service.process_vnpay_ipn(params)
    assert response["RspCode"] == "01"


async def test_vnpay_ipn_amount_mismatch(finance_service):
    """IPN with mismatched amount returns RspCode 04."""
    import hmac
    import hashlib

    # Create a real PENDING topup transaction
    vnpay_url = await finance_service.initiate_topup(
        user_id="u-sad-8",
        amount_coins=50,
        client_ip="127.0.0.1",
    )
    import urllib.parse

    parsed = urllib.parse.urlparse(vnpay_url)
    txn_ref = urllib.parse.parse_qs(parsed.query)["vnp_TxnRef"][0]

    # Send IPN with wrong amount (50,000 VND → 5,000,000 cents; but we use 1,000 cents = wrong)
    params = {
        "vnp_TmnCode": "TMN_TEST",
        "vnp_Amount": "100",  # deliberately wrong
        "vnp_TxnRef": txn_ref,
        "vnp_ResponseCode": "00",
    }
    sorted_params = sorted(params.items())
    raw_query = "&".join(f"{k}={v}" for k, v in sorted_params)
    secure_hash = (
        hmac.new(
            key=b"SECRET_TEST",
            msg=raw_query.encode("utf-8"),
            digestmod=hashlib.sha512,
        )
        .hexdigest()
        .upper()
    )
    params["vnp_SecureHash"] = secure_hash

    response = await finance_service.process_vnpay_ipn(params)
    assert response["RspCode"] == "04"


async def test_vnpay_ipn_failed_payment_marks_transaction_failed(finance_service):
    """IPN with vnp_ResponseCode != '00' marks transaction as FAILED and still returns RspCode 00."""
    import hmac
    import hashlib

    vnpay_url = await finance_service.initiate_topup(
        user_id="u-sad-9",
        amount_coins=10,
        client_ip="127.0.0.1",
    )
    import urllib.parse

    parsed = urllib.parse.urlparse(vnpay_url)
    txn_ref = urllib.parse.parse_qs(parsed.query)["vnp_TxnRef"][0]

    params = {
        "vnp_TmnCode": "TMN_TEST",
        "vnp_Amount": "1000000",  # 10 coins * 1000 VND * 100 cents
        "vnp_TxnRef": txn_ref,
        "vnp_ResponseCode": "24",  # User cancelled
    }
    sorted_params = sorted(params.items())
    raw_query = "&".join(f"{k}={v}" for k, v in sorted_params)
    secure_hash = (
        hmac.new(
            key=b"SECRET_TEST",
            msg=raw_query.encode("utf-8"),
            digestmod=hashlib.sha512,
        )
        .hexdigest()
        .upper()
    )
    params["vnp_SecureHash"] = secure_hash

    response = await finance_service.process_vnpay_ipn(params)
    assert response["RspCode"] == "00"  # VNPay standard: acknowledge receipt

    # Transaction must be FAILED
    txn = await finance_service.transaction_repo.find_by_id(txn_ref)
    assert txn.status == "FAILED"

    # Wallet must NOT be credited — initiate_topup does NOT lazy-create the wallet,
    # so for a failed payment the wallet is never created.
    wallet = await finance_service.wallet_repo.find_by_user_id("u-sad-9")
    assert wallet is None


# ---------------------------------------------------------------------------
# 5. gRPC servicer failure event publication sad paths
# ---------------------------------------------------------------------------


class MockGRPCContext:
    def __init__(self):
        self.code = None
        self.details = None

    def set_code(self, code):
        self.code = code

    def set_details(self, details):
        self.details = details


class MockRequest:
    def __init__(self, **kwargs):
        for k, v in kwargs.items():
            setattr(self, k, v)


@pytest.fixture
async def test_session_factory():
    engine = create_async_engine("sqlite+aiosqlite:///:memory:", echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    SessionLocal = async_sessionmaker(
        autocommit=False, autoflush=False, bind=engine, class_=AsyncSession
    )
    yield SessionLocal
    await engine.dispose()


async def test_grpc_freeze_coin_insufficient_balance_publishes_escrow_failed(
    test_session_factory,
):
    servicer = FinanceServiceServicer(test_session_factory)
    context = MockGRPCContext()
    request = MockRequest(user_id="u-sad-grpc-1", amount=100, booking_id="b-sad-grpc-1")

    # Call servicer (should fail because wallet has 0 balance)
    await servicer.FreezeCoin(request, context)
    assert context.code == grpc.StatusCode.FAILED_PRECONDITION
    assert "Insufficient available balance" in context.details

    # Verify EscrowFailed event was published to the outbox database table
    async with test_session_factory() as session:
        stmt = select(OutboxModel).filter(
            OutboxModel.event_type == "finance.escrow-failed.v1"
        )
        result = await session.execute(stmt)
        outbox_events = result.scalars().all()
        assert len(outbox_events) == 1

        import json

        payload = json.loads(outbox_events[0].payload)
        assert payload["bookingId"] == "b-sad-grpc-1"
        assert payload["clientId"] == "u-sad-grpc-1"
        assert "Insufficient available balance" in payload["reason"]


async def test_grpc_refund_escrow_not_found_publishes_refund_failed(
    test_session_factory,
):
    servicer = FinanceServiceServicer(test_session_factory)
    context = MockGRPCContext()
    request = MockRequest(
        booking_id="b-sad-grpc-2", client_id="u-sad-grpc-2", refund_amount=100
    )

    # Call servicer (should fail because escrow does not exist)
    await servicer.RefundEscrow(request, context)
    assert context.code == grpc.StatusCode.NOT_FOUND

    # Verify RefundFailed event was published to outbox
    async with test_session_factory() as session:
        stmt = select(OutboxModel).filter(
            OutboxModel.event_type == "finance.refund-failed.v1"
        )
        result = await session.execute(stmt)
        outbox_events = result.scalars().all()
        assert len(outbox_events) == 1

        import json

        payload = json.loads(outbox_events[0].payload)
        assert payload["bookingId"] == "b-sad-grpc-2"
        assert payload["clientId"] == "u-sad-grpc-2"
        assert "Escrow not found" in payload["reason"]


async def test_grpc_refund_escrow_empty_client_resolves_fallback_and_publishes_refund_failed(
    test_session_factory,
):
    # Seed a reservation transaction first
    async with test_session_factory() as session:
        # Create a transaction model representing a booking reservation
        txn = TransactionModel(
            transaction_id="t-res-1",
            user_id="u-resolved-client",
            amount=150,
            type="BOOKING_RESERVATION",
            status="SUCCESS",
            reference_id="b-sad-grpc-3",
        )
        session.add(txn)
        await session.commit()

    servicer = FinanceServiceServicer(test_session_factory)
    context = MockGRPCContext()
    # client_id is empty, refund_amount is 0. Both should fall back to reservation transaction!
    request = MockRequest(booking_id="b-sad-grpc-3", client_id="", refund_amount=0)

    # Call servicer (should fail because escrow does not exist, but client_id and refund_amount should be resolved first!)
    await servicer.RefundEscrow(request, context)
    assert context.code == grpc.StatusCode.NOT_FOUND

    # Verify RefundFailed event was published to outbox with the correct resolved client_id!
    async with test_session_factory() as session:
        stmt = select(OutboxModel).filter(
            OutboxModel.event_type == "finance.refund-failed.v1"
        )
        result = await session.execute(stmt)
        outbox_events = result.scalars().all()
        assert len(outbox_events) == 1

        import json

        payload = json.loads(outbox_events[0].payload)
        assert payload["bookingId"] == "b-sad-grpc-3"
        assert payload["clientId"] == "u-resolved-client"  # Resolved!
        assert "Escrow not found" in payload["reason"]
