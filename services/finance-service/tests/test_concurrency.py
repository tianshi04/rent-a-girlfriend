import os
import uuid
import hmac
import hashlib
import asyncio
import pytest
from typing import List
from sqlalchemy import event
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

from internal.domain.vo import Money
from internal.domain.events import DomainEvent
from internal.domain.aggregate.wallet import Wallet
from internal.domain.aggregate.escrow import Escrow
from internal.application.port import IEventPublisher
from internal.application.command.finance import FinanceCommandService
from internal.infrastructure.persistence.models import Base
from internal.infrastructure.persistence.repositories import (
    WalletRepository,
    EscrowRepository,
    TransactionRepository,
)
from internal.infrastructure.payment.vnpay import VNPayAdapter

HASH_SECRET = "CONCURRENCY_TEST_SECRET"
TMN_CODE = "CONCURRENCY_TMN"


class MockEventPublisher(IEventPublisher):
    def __init__(self):
        self.events: List[DomainEvent] = []

    def publish(self, event: DomainEvent) -> None:
        self.events.append(event)

    def clear(self):
        self.events = []


def build_signed_ipn(params: dict, secret: str) -> dict:
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


@pytest.fixture
async def db_engine():
    # SQLite on-disk DB to ensure full concurrency capabilities across multiple connections/sessions
    db_file = "test_concurrency.db"
    if os.path.exists(db_file):
        try:
            os.remove(db_file)
        except Exception:
            pass

    engine = create_async_engine(f"sqlite+aiosqlite:///{db_file}", echo=False)

    # Register listeners to configure SQLite for IMMEDIATE transaction locking behavior.
    # This emulates the pessimistic locking behavior of SELECT ... FOR UPDATE in PostgreSQL.
    @event.listens_for(engine.sync_engine, "connect")
    def set_sqlite_pragma(dbapi_connection, connection_record):
        cursor = dbapi_connection.cursor()
        cursor.execute("PRAGMA busy_timeout = 5000")
        cursor.close()

    @event.listens_for(engine.sync_engine, "begin")
    def do_begin(conn):
        conn.exec_driver_sql("BEGIN IMMEDIATE")

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    await engine.dispose()

    # Clean up the file
    if os.path.exists(db_file):
        try:
            os.remove(db_file)
        except Exception:
            pass


@pytest.fixture
async def session_factory(db_engine):
    return async_sessionmaker(
        autocommit=False, autoflush=False, bind=db_engine, class_=AsyncSession
    )


@pytest.fixture
def mock_vnpay_adapter():
    return VNPayAdapter(
        tmn_code=TMN_CODE,
        hash_secret=HASH_SECRET,
        payment_url="https://sandbox.vnpayment.vn",
        return_url="http://localhost:8080/return",
    )


@pytest.fixture
def mock_publisher():
    return MockEventPublisher()


async def test_concurrent_freeze_coin(
    session_factory, mock_vnpay_adapter, mock_publisher
):
    # Pre-seed wallet with 100 coins
    user_id = "concurrent-user-1"
    async with session_factory() as db:
        wallet_repo = WalletRepository(db)
        wallet = Wallet.create(str(uuid.uuid4()), user_id)
        wallet.available_balance = Money(100)
        await wallet_repo.save(wallet)
        await db.commit()

    async def run_freeze(amount, booking_id):
        async with session_factory() as db:
            svc = FinanceCommandService(
                wallet_repo=WalletRepository(db),
                escrow_repo=EscrowRepository(db),
                transaction_repo=TransactionRepository(db),
                event_publisher=mock_publisher,
                vnpay_adapter=mock_vnpay_adapter,
            )
            try:
                await svc.freeze_coin(
                    user_id=user_id, amount=amount, booking_id=booking_id
                )
                await db.commit()
                return "SUCCESS"
            except Exception as e:
                await db.rollback()
                return str(e) or e.__class__.__name__

    # Execute two freeze_coin tasks concurrently on the same wallet
    results = await asyncio.gather(
        run_freeze(80, "booking-a"), run_freeze(80, "booking-b")
    )

    # Verify that one succeeded and the other failed with InsufficientBalanceError
    assert "SUCCESS" in results
    errors = [r for r in results if r != "SUCCESS"]
    assert len(errors) == 1
    assert "Insufficient" in errors[0]

    # Verify final wallet balance: 20 available, 80 frozen (exactly one reservation succeeded)
    async with session_factory() as db:
        wallet_repo = WalletRepository(db)
        w = await wallet_repo.find_by_user_id(user_id)
        assert w.available_balance.amount == 20
        assert w.frozen_balance.amount == 80


async def test_concurrent_vnpay_ipn(
    session_factory, mock_vnpay_adapter, mock_publisher
):
    user_id = "concurrent-user-2"

    # 1. Create a PENDING transaction
    async with session_factory() as db:
        svc = FinanceCommandService(
            wallet_repo=WalletRepository(db),
            escrow_repo=EscrowRepository(db),
            transaction_repo=TransactionRepository(db),
            event_publisher=mock_publisher,
            vnpay_adapter=mock_vnpay_adapter,
        )
        await svc.get_or_create_wallet(user_id)
        url = await svc.initiate_topup(
            user_id=user_id, amount_coins=50, client_ip="127.0.0.1"
        )
        await db.commit()

    import urllib.parse

    parsed = urllib.parse.urlparse(url)
    txn_ref = urllib.parse.parse_qs(parsed.query)["vnp_TxnRef"][0]

    # 2. Build signed IPN params
    ipn_params = build_signed_ipn(
        {
            "vnp_TmnCode": TMN_CODE,
            "vnp_Amount": "5000000",  # 50 coins * 1000 * 100 cents
            "vnp_TxnRef": txn_ref,
            "vnp_ResponseCode": "00",
        },
        HASH_SECRET,
    )

    async def run_ipn():
        async with session_factory() as db:
            svc = FinanceCommandService(
                wallet_repo=WalletRepository(db),
                escrow_repo=EscrowRepository(db),
                transaction_repo=TransactionRepository(db),
                event_publisher=mock_publisher,
                vnpay_adapter=mock_vnpay_adapter,
            )
            try:
                res = await svc.process_vnpay_ipn(ipn_params)
                await db.commit()
                return res
            except Exception as e:
                await db.rollback()
                return {"error": str(e)}

    # Run concurrently
    results = await asyncio.gather(run_ipn(), run_ipn())

    # One response must be 00 (success), the other must be 02 (already confirmed)
    rsp_codes = [r.get("RspCode") for r in results]
    assert "00" in rsp_codes
    assert "02" in rsp_codes

    # Verify wallet has only been credited once: 50 coins
    async with session_factory() as db:
        wallet_repo = WalletRepository(db)
        w = await wallet_repo.find_by_user_id(user_id)
        assert w.available_balance.amount == 50


async def test_concurrent_escrow_payout(
    session_factory, mock_vnpay_adapter, mock_publisher
):
    companion_id = "companion-1"
    booking_id = "booking-1"

    # Pre-seed escrow as HELD and companion wallet to prevent UNIQUE constraint race condition
    async with session_factory() as db:
        escrow_repo = EscrowRepository(db)
        escrow = Escrow.create(str(uuid.uuid4()), booking_id, Money(100))
        await escrow_repo.save(escrow)

        wallet_repo = WalletRepository(db)
        companion_wallet = Wallet.create(str(uuid.uuid4()), companion_id)
        await wallet_repo.save(companion_wallet)

        await db.commit()

    async def run_payout():
        async with session_factory() as db:
            svc = FinanceCommandService(
                wallet_repo=WalletRepository(db),
                escrow_repo=EscrowRepository(db),
                transaction_repo=TransactionRepository(db),
                event_publisher=mock_publisher,
                vnpay_adapter=mock_vnpay_adapter,
            )
            try:
                await svc.process_payout(
                    booking_id=booking_id,
                    companion_id=companion_id,
                    commission_rate=0.1,
                )
                await db.commit()
                return "SUCCESS"
            except Exception as e:
                await db.rollback()
                return str(e) or e.__class__.__name__

    # Run concurrently
    results = await asyncio.gather(run_payout(), run_payout())

    # One must succeed, other must fail (e.g. InvalidEscrowStatusTransitionError)
    assert "SUCCESS" in results
    errors = [r for r in results if r != "SUCCESS"]
    assert len(errors) == 1
    assert "Invalid" in errors[0] or "Cannot transition" in errors[0]

    # Verify companion wallet was credited exactly once (100 - 10 = 90 coins)
    async with session_factory() as db:
        wallet_repo = WalletRepository(db)
        w = await wallet_repo.find_by_user_id(companion_id)
        assert w.available_balance.amount == 90


async def test_concurrent_escrow_refund(
    session_factory, mock_vnpay_adapter, mock_publisher
):
    client_id = "client-1"
    booking_id = "booking-2"

    # Pre-seed escrow as HELD and client wallet to prevent UNIQUE constraint race condition
    async with session_factory() as db:
        escrow_repo = EscrowRepository(db)
        escrow = Escrow.create(str(uuid.uuid4()), booking_id, Money(100))
        await escrow_repo.save(escrow)

        wallet_repo = WalletRepository(db)
        client_wallet = Wallet.create(str(uuid.uuid4()), client_id)
        await wallet_repo.save(client_wallet)

        await db.commit()

    async def run_refund():
        async with session_factory() as db:
            svc = FinanceCommandService(
                wallet_repo=WalletRepository(db),
                escrow_repo=EscrowRepository(db),
                transaction_repo=TransactionRepository(db),
                event_publisher=mock_publisher,
                vnpay_adapter=mock_vnpay_adapter,
            )
            try:
                await svc.refund_escrow(
                    booking_id=booking_id, client_id=client_id, refund_amount=100
                )
                await db.commit()
                return "SUCCESS"
            except Exception as e:
                await db.rollback()
                return str(e) or e.__class__.__name__

    # Run concurrently
    results = await asyncio.gather(run_refund(), run_refund())

    # One must succeed, other must fail
    assert "SUCCESS" in results
    errors = [r for r in results if r != "SUCCESS"]
    assert len(errors) == 1
    assert "Invalid" in errors[0] or "Cannot transition" in errors[0]

    # Verify client wallet was credited exactly once (100 coins)
    async with session_factory() as db:
        wallet_repo = WalletRepository(db)
        w = await wallet_repo.find_by_user_id(client_id)
        assert w.available_balance.amount == 100
