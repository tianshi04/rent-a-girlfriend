import pytest
import hmac
import hashlib
from typing import List

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


# 1. Mock Event Publisher
class MockEventPublisher(IEventPublisher):
    def __init__(self):
        self.events: List[DomainEvent] = []

    def publish(self, event: DomainEvent) -> None:
        self.events.append(event)

    def clear(self):
        self.events = []


# 2. Database test fixtures


@pytest.fixture
def mock_vnpay_adapter():
    return VNPayAdapter(
        tmn_code="TMN_123",
        hash_secret="SECRET_123",
        payment_url="https://sandbox.vnpayment.vn",
        return_url="http://localhost:8080/return",
    )


@pytest.fixture
def mock_publisher():
    return MockEventPublisher()


@pytest.fixture
def finance_service(db_session, mock_vnpay_adapter, mock_publisher):
    wallet_repo = WalletRepository(db_session)
    escrow_repo = EscrowRepository(db_session)
    transaction_repo = TransactionRepository(db_session)

    return FinanceCommandService(
        wallet_repo=wallet_repo,
        escrow_repo=escrow_repo,
        transaction_repo=transaction_repo,
        event_publisher=mock_publisher,
        vnpay_adapter=mock_vnpay_adapter,
    )


# 3. Test Cases
async def test_wallet_lazy_onboarding(finance_service):
    # Retrieve wallet for a new user, should auto-create
    user_id = "user-1"
    wallet = await finance_service.get_or_create_wallet(user_id)
    assert wallet is not None
    assert wallet.user_id == user_id
    assert wallet.available_balance.amount == 0


async def test_wallet_active_onboarding_idempotency(finance_service):
    user_id = "user-2"

    # Onboard once
    await finance_service.create_wallet_onboard(user_id)
    w1 = await finance_service.wallet_repo.find_by_user_id(user_id)
    assert w1 is not None

    # Onboard again (idempotent skip)
    await finance_service.create_wallet_onboard(user_id)
    w2 = await finance_service.wallet_repo.find_by_user_id(user_id)
    assert w2.wallet_id == w1.wallet_id


async def test_booking_reservation_flow(finance_service, mock_publisher):
    client_id = "client-1"
    companion_id = "companion-1"
    booking_id = "booking-1"

    # Onboard & deposit coins to client wallet
    client_wallet = await finance_service.get_or_create_wallet(client_id)
    client_wallet.available_balance = client_wallet.available_balance.add(
        Money(100)
    )  # Give 100 coins
    await finance_service.wallet_repo.save(client_wallet)

    # --- STEP 1: FREEZE COIN ---
    txn_id = await finance_service.freeze_coin(
        user_id=client_id, amount=80, booking_id=booking_id
    )
    assert txn_id is not None

    # Verify balances
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 20
    assert w.frozen_balance.amount == 80

    # Verify transaction log in DB
    txn = await finance_service.transaction_repo.find_by_id(txn_id)
    assert txn.status == "PENDING"
    assert txn.type == "BOOKING_RESERVATION"
    assert txn.amount.amount == 80

    # Verify outbox event
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "CoinsFrozen"
    mock_publisher.clear()

    # --- STEP 2: TRANSFER TO ESCROW ---
    escrow_id = await finance_service.transfer_to_escrow(
        user_id=client_id, amount=80, booking_id=booking_id
    )
    assert escrow_id is not None

    # Verify wallet balances
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 20
    assert w.frozen_balance.amount == 0  # Deducted

    # Verify Escrow in DB (status HELD)
    escrow = await finance_service.escrow_repo.find_by_id(escrow_id)
    assert escrow.status == "HELD"
    assert escrow.amount.amount == 80
    assert escrow.booking_id == booking_id

    # Verify transaction log update
    txn_escrow = await finance_service.transaction_repo.find_by_id(txn_id)
    assert txn_escrow.status == "SUCCESS"

    # Verify outbox event
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "EscrowCreated"
    mock_publisher.clear()

    # --- STEP 3: RELEASE ESCROW PAYOUT ---
    payout_txn_id = await finance_service.process_payout(
        booking_id=booking_id,
        companion_id=companion_id,
        commission_rate=0.1,  # 10% commission
    )
    assert payout_txn_id is not None

    # 80 * 0.1 = 8 commission, net = 72 coins
    escrow_final = await finance_service.escrow_repo.find_by_id(escrow_id)
    assert escrow_final.status == "PAID_OUT"

    # Verify companion got net coins
    comp_wallet = await finance_service.wallet_repo.find_by_user_id(companion_id)
    assert comp_wallet.available_balance.amount == 72

    # Verify outbox event
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "PayoutProcessed"
    assert mock_publisher.events[0].commission_amount == 8
    assert mock_publisher.events[0].net_amount == 72


async def test_escrow_refund_flow(finance_service, mock_publisher):
    client_id = "client-2"
    booking_id = "booking-2"

    # Set up active Escrow (status HELD)
    client_wallet = await finance_service.get_or_create_wallet(client_id)
    client_wallet.available_balance = client_wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(client_wallet)

    await finance_service.freeze_coin(client_id, 100, booking_id)
    escrow_id = await finance_service.transfer_to_escrow(client_id, 100, booking_id)
    mock_publisher.clear()

    # REFUND ESCROW
    refund_txn_id = await finance_service.refund_escrow(
        booking_id=booking_id, client_id=client_id, refund_amount=100
    )
    assert refund_txn_id is not None

    # Verify escrow status
    escrow = await finance_service.escrow_repo.find_by_id(escrow_id)
    assert escrow.status == "REFUNDED"

    # Verify client wallet balance refunded
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 100

    # Verify outbox event
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "EscrowRefunded"
    assert mock_publisher.events[0].refund_amount == 100


async def test_vnpay_topup_and_ipn_flow(finance_service, mock_publisher):
    user_id = "client-3"
    amount_coins = 50  # 50,000 VND

    # 1. INITIATE TOPUP
    vnpay_url = await finance_service.initiate_topup(
        user_id=user_id, amount_coins=amount_coins, client_ip="127.0.0.1"
    )
    assert "vnp_TxnRef" in vnpay_url

    # Extract txn_ref from generated payment URL
    import urllib.parse

    parsed_url = urllib.parse.urlparse(vnpay_url)
    query_params = urllib.parse.parse_qs(parsed_url.query)
    txn_ref = query_params["vnp_TxnRef"][0]

    # Verify pending transaction in DB
    txn = await finance_service.transaction_repo.find_by_id(txn_ref)
    assert txn.status == "PENDING"
    assert txn.type == "TOPUP"
    assert txn.amount.amount == 50

    # 2. VNPay IPN Webhook simulation
    # Construct exact query parameters returned by VNPay Sandbox
    ipn_params = {
        "vnp_TmnCode": "TMN_123",
        "vnp_Amount": "5000000",  # Cent/VND: 50,000 VND * 100
        "vnp_TxnRef": txn_ref,
        "vnp_ResponseCode": "00",  # Success code
        "vnp_TransactionNo": "12345678",
        "vnp_CardType": "NCB",
    }

    # Build raw secure hash signature using mock credentials
    sorted_params = sorted(ipn_params.items())
    raw_query = "&".join(f"{k}={v}" for k, v in sorted_params)
    secure_hash = (
        hmac.new(
            key=b"SECRET_123", msg=raw_query.encode("utf-8"), digestmod=hashlib.sha512
        )
        .hexdigest()
        .upper()
    )

    ipn_params["vnp_SecureHash"] = secure_hash

    # Call IPN Webhook handler
    response = await finance_service.process_vnpay_ipn(ipn_params)
    assert response["RspCode"] == "00"

    # Verify wallet balances topped up
    w = await finance_service.wallet_repo.find_by_user_id(user_id)
    assert w.available_balance.amount == 50

    # Verify Transaction marked SUCCESS
    txn_success = await finance_service.transaction_repo.find_by_id(txn_ref)
    assert txn_success.status == "SUCCESS"

    # Verify outbox event WalletToppedUp published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "WalletToppedUp"
    assert mock_publisher.events[0].amount == 50
    assert mock_publisher.events[0].vnpay_amount_vnd == 50000
    mock_publisher.clear()

    # 3. IDEMPOTENCY TEST: Call IPN Webhook again with same txn_ref
    dup_response = await finance_service.process_vnpay_ipn(ipn_params)
    assert dup_response["RspCode"] == "02"  # Order already confirmed

    # Balances must remain unchanged
    w_dup = await finance_service.wallet_repo.find_by_user_id(user_id)
    assert w_dup.available_balance.amount == 50


async def test_outbox_publisher_worker_cloudevent():
    from sqlalchemy.ext.asyncio import (
        create_async_engine,
        AsyncSession,
        async_sessionmaker,
    )
    from internal.domain import events as domain_events
    from internal.infrastructure.persistence.models import Base, OutboxModel
    from internal.infrastructure.broker.outbox_publisher import (
        DatabaseEventPublisher,
        OutboxPublisherWorker,
    )
    from sqlalchemy import select

    # 1. Setup in-memory SQLite for this test
    engine = create_async_engine("sqlite+aiosqlite:///:memory:", echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    SessionLocal = async_sessionmaker(
        autocommit=False, autoflush=False, bind=engine, class_=AsyncSession
    )

    # 2. Insert event via DatabaseEventPublisher
    async with SessionLocal() as session:
        publisher = DatabaseEventPublisher(session)
        event = domain_events.WalletToppedUp(
            wallet_id="wallet-123",
            user_id="user-123",
            amount=50,
            vnpay_amount_vnd=50000,
        )
        publisher.publish(event)
        await session.commit()

    # Get the generated event_id from outbox table
    async with SessionLocal() as session:
        stmt = select(OutboxModel)
        res = await session.execute(stmt)
        outbox_record = res.scalar()
        event_id = outbox_record.event_id

    # 3. Mock Kafka Producer
    published_events = []

    class MockKafkaProducer:
        async def send_and_wait(self, topic, key, value):
            published_events.append((topic, key, value))

    worker = OutboxPublisherWorker(
        session_factory=SessionLocal,
        kafka_brokers="localhost:9092",
        topic="finance.events",
    )
    worker.producer = MockKafkaProducer()

    # 4. Run worker batch process
    await worker._process_batch()

    # 5. Verify published message
    assert len(published_events) == 1
    topic, key, value = published_events[0]
    assert topic == "finance.events"
    assert key == b"user-123"

    assert value["specversion"] == "1.0"
    assert value["id"] == event_id
    assert value["type"] == "finance.wallet-topped-up.v1"
    assert value["datacontenttype"] == "application/json"
    assert value["correlationid"] == event_id
    assert "extensions" not in value or "correlationId" not in value.get(
        "extensions", {}
    )
    assert value["data"]["userId"] == "user-123"
    assert value["data"]["amount"] == "50"

    # 6. Verify marked as processed in DB
    async with SessionLocal() as session:
        stmt = select(OutboxModel)
        res = await session.execute(stmt)
        record = res.scalar()
        assert record.processed is True

    await engine.dispose()


async def test_booking_event_listener_success(
    finance_service, db_session, mock_publisher, monkeypatch
):
    import asyncio

    finance_service.session = db_session
    # Set up client wallet with 100 coins
    client_id = "client-event-1"
    booking_id = "booking-event-1"

    client_wallet = await finance_service.get_or_create_wallet(client_id)
    client_wallet.available_balance = client_wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(client_wallet)
    await db_session.commit()

    # Mock CloudEvent payload for booking.booking-requested.v1
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-123",
        "type": "booking.booking-requested.v1",
        "datacontenttype": "application/json",
        "data": {"bookingId": booking_id, "clientId": client_id, "price": 80},
    }

    class MockMessage:
        def __init__(self, value):
            self.value = value

    class MockConsumer:
        def __init__(self, *args, **kwargs):
            self.messages = [MockMessage(mock_msg_value)]

        async def start(self):
            pass

        async def stop(self):
            pass

        def __aiter__(self):
            return self

        async def __anext__(self):
            if not self.messages:
                raise asyncio.CancelledError()
            return self.messages.pop(0)

    import main as server_main

    monkeypatch.setattr(server_main, "AIOKafkaConsumer", MockConsumer)

    class MockSessionContext:
        def __init__(self, session):
            self.session = session

        async def __aenter__(self):
            return self.session

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            pass

    monkeypatch.setattr(
        server_main, "SessionLocal", lambda: MockSessionContext(db_session)
    )
    monkeypatch.setattr(server_main, "bootstrap_services", lambda sess: finance_service)

    # Run the listener (it will process the message and then raise CancelledError)
    await server_main.run_booking_event_listener()

    # Verify the client wallet balance after processing
    db_session.expire_all()
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 20
    assert w.frozen_balance.amount == 80

    # Verify transaction log in DB
    txn = await finance_service.transaction_repo.find_by_reference_id(
        booking_id, "BOOKING_RESERVATION"
    )
    assert txn is not None
    assert txn.status == "PENDING"
    assert txn.amount.amount == 80

    # Verify outbox event (CoinsFrozen) is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "CoinsFrozen"
    assert mock_publisher.events[0].amount == 80


async def test_booking_event_listener_insufficient_funds(
    finance_service, db_session, mock_publisher, monkeypatch
):
    import asyncio

    finance_service.session = db_session
    client_id = "client-event-2"
    booking_id = "booking-event-2"

    client_wallet = await finance_service.get_or_create_wallet(client_id)
    client_wallet.available_balance = client_wallet.available_balance.add(
        Money(50)
    )  # Only 50 coins
    await finance_service.wallet_repo.save(client_wallet)
    await db_session.commit()

    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-456",
        "type": "booking.booking-requested.v1",
        "data": {
            "bookingId": booking_id,
            "clientId": client_id,
            "price": 80,  # Needs 80
        },
    }

    class MockMessage:
        def __init__(self, value):
            self.value = value

    class MockConsumer:
        def __init__(self, *args, **kwargs):
            self.messages = [MockMessage(mock_msg_value)]

        async def start(self):
            pass

        async def stop(self):
            pass

        def __aiter__(self):
            return self

        async def __anext__(self):
            if not self.messages:
                raise asyncio.CancelledError()
            return self.messages.pop(0)

    import main as server_main

    monkeypatch.setattr(server_main, "AIOKafkaConsumer", MockConsumer)

    class MockSessionContext:
        def __init__(self, session):
            self.session = session

        async def __aenter__(self):
            return self.session

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            pass

    monkeypatch.setattr(
        server_main, "SessionLocal", lambda: MockSessionContext(db_session)
    )
    monkeypatch.setattr(server_main, "bootstrap_services", lambda sess: finance_service)

    # Run the listener
    await server_main.run_booking_event_listener()

    # Verify the client wallet balance is unchanged
    db_session.expire_all()
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 50
    assert w.frozen_balance.amount == 0

    # Verify CoinsFreezeFailed event is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "CoinsFreezeFailed"
    assert mock_publisher.events[0].amount == 80
    assert mock_publisher.events[0].booking_id == booking_id


async def test_transfer_to_escrow_event_listener_success(
    finance_service, db_session, mock_publisher, monkeypatch
):
    import asyncio

    finance_service.session = db_session
    client_id = "client-event-3"
    booking_id = "booking-event-3"

    # Set up client wallet with 20 available, 80 frozen coins
    client_wallet = await finance_service.get_or_create_wallet(client_id)
    client_wallet.available_balance = client_wallet.available_balance.add(Money(20))
    client_wallet.frozen_balance = client_wallet.frozen_balance.add(Money(80))
    await finance_service.wallet_repo.save(client_wallet)
    await db_session.commit()
    mock_publisher.clear()

    # Mock CloudEvent payload for finance.transfer-to-escrow.v1
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-789",
        "type": "finance.transfer-to-escrow.v1",
        "data": {"bookingId": booking_id, "userId": client_id, "amount": 80},
    }

    class MockMessage:
        def __init__(self, value):
            self.value = value

    class MockConsumer:
        def __init__(self, *args, **kwargs):
            self.messages = [MockMessage(mock_msg_value)]

        async def start(self):
            pass

        async def stop(self):
            pass

        def __aiter__(self):
            return self

        async def __anext__(self):
            if not self.messages:
                raise asyncio.CancelledError()
            return self.messages.pop(0)

    import main as server_main

    monkeypatch.setattr(server_main, "AIOKafkaConsumer", MockConsumer)

    class MockSessionContext:
        def __init__(self, session):
            self.session = session

        async def __aenter__(self):
            return self.session

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            pass

    monkeypatch.setattr(
        server_main, "SessionLocal", lambda: MockSessionContext(db_session)
    )
    monkeypatch.setattr(server_main, "bootstrap_services", lambda sess: finance_service)

    # Run the listener
    await server_main.run_booking_event_listener()

    # Verify client wallet has 0 frozen coins
    db_session.expire_all()
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 20
    assert w.frozen_balance.amount == 0

    # Verify escrow is created in HELD status
    escrow = await finance_service.escrow_repo.find_by_booking_id(booking_id)
    assert escrow is not None
    assert escrow.status == "HELD"
    assert escrow.amount.amount == 80

    # Verify outbox event (EscrowCreated) is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "EscrowCreated"
    assert mock_publisher.events[0].amount == 80
    assert mock_publisher.events[0].booking_id == booking_id


async def test_transfer_to_escrow_event_listener_wallet_not_found(
    finance_service, db_session, mock_publisher, monkeypatch
):
    import asyncio

    finance_service.session = db_session
    client_id = "non-existent-client"
    booking_id = "booking-event-4"
    mock_publisher.clear()

    # Mock CloudEvent payload for finance.transfer-to-escrow.v1
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-101",
        "type": "finance.transfer-to-escrow.v1",
        "data": {"bookingId": booking_id, "userId": client_id, "amount": 80},
    }

    class MockMessage:
        def __init__(self, value):
            self.value = value

    class MockConsumer:
        def __init__(self, *args, **kwargs):
            self.messages = [MockMessage(mock_msg_value)]

        async def start(self):
            pass

        async def stop(self):
            pass

        def __aiter__(self):
            return self

        async def __anext__(self):
            if not self.messages:
                raise asyncio.CancelledError()
            return self.messages.pop(0)

    import main as server_main

    monkeypatch.setattr(server_main, "AIOKafkaConsumer", MockConsumer)

    class MockSessionContext:
        def __init__(self, session):
            self.session = session

        async def __aenter__(self):
            return self.session

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            pass

    monkeypatch.setattr(
        server_main, "SessionLocal", lambda: MockSessionContext(db_session)
    )
    monkeypatch.setattr(server_main, "bootstrap_services", lambda sess: finance_service)

    # Run the listener
    await server_main.run_booking_event_listener()

    # Verify EscrowFailed event is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "EscrowFailed"
    assert mock_publisher.events[0].booking_id == booking_id
    assert mock_publisher.events[0].client_id == client_id


async def test_refund_escrow_event_listener_success(
    finance_service, db_session, mock_publisher, monkeypatch
):
    import asyncio

    finance_service.session = db_session
    client_id = "client-event-5"
    booking_id = "booking-event-5"

    # Set up client wallet and escrow
    client_wallet = await finance_service.get_or_create_wallet(client_id)
    client_wallet.available_balance = client_wallet.available_balance.add(Money(100))
    client_wallet.frozen_balance = client_wallet.frozen_balance.add(Money(80))
    await finance_service.wallet_repo.save(client_wallet)
    await db_session.commit()

    # Create escrow
    await finance_service.transfer_to_escrow(client_id, 80, booking_id)
    await db_session.commit()
    mock_publisher.clear()

    # Mock CloudEvent payload for finance.refund-escrow.v1
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-202",
        "type": "finance.refund-escrow.v1",
        "data": {"bookingId": booking_id, "clientId": client_id, "refundAmount": 80},
    }

    class MockMessage:
        def __init__(self, value):
            self.value = value

    class MockConsumer:
        def __init__(self, *args, **kwargs):
            self.messages = [MockMessage(mock_msg_value)]

        async def start(self):
            pass

        async def stop(self):
            pass

        def __aiter__(self):
            return self

        async def __anext__(self):
            if not self.messages:
                raise asyncio.CancelledError()
            return self.messages.pop(0)

    import main as server_main

    monkeypatch.setattr(server_main, "AIOKafkaConsumer", MockConsumer)

    class MockSessionContext:
        def __init__(self, session):
            self.session = session

        async def __aenter__(self):
            return self.session

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            pass

    monkeypatch.setattr(
        server_main, "SessionLocal", lambda: MockSessionContext(db_session)
    )
    monkeypatch.setattr(server_main, "bootstrap_services", lambda sess: finance_service)

    # Run the listener
    await server_main.run_booking_event_listener()

    # Verify client wallet has refund deposited (100 + 80 [refund] = 180)
    db_session.expire_all()
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 180

    # Verify escrow status is REFUNDED
    escrow = await finance_service.escrow_repo.find_by_booking_id(booking_id)
    assert escrow.status == "REFUNDED"

    # Verify EscrowRefunded event is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "EscrowRefunded"
    assert mock_publisher.events[0].booking_id == booking_id
    assert mock_publisher.events[0].refund_amount == 80


async def test_refund_escrow_event_listener_escrow_not_found(
    finance_service, db_session, mock_publisher, monkeypatch
):
    import asyncio

    finance_service.session = db_session
    client_id = "client-event-6"
    booking_id = "non-existent-escrow"
    mock_publisher.clear()

    # Mock CloudEvent payload for finance.refund-escrow.v1
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-303",
        "type": "finance.refund-escrow.v1",
        "data": {"bookingId": booking_id, "clientId": client_id, "refundAmount": 80},
    }

    class MockMessage:
        def __init__(self, value):
            self.value = value

    class MockConsumer:
        def __init__(self, *args, **kwargs):
            self.messages = [MockMessage(mock_msg_value)]

        async def start(self):
            pass

        async def stop(self):
            pass

        def __aiter__(self):
            return self

        async def __anext__(self):
            if not self.messages:
                raise asyncio.CancelledError()
            return self.messages.pop(0)

    import main as server_main

    monkeypatch.setattr(server_main, "AIOKafkaConsumer", MockConsumer)

    class MockSessionContext:
        def __init__(self, session):
            self.session = session

        async def __aenter__(self):
            return self.session

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            pass

    monkeypatch.setattr(
        server_main, "SessionLocal", lambda: MockSessionContext(db_session)
    )
    monkeypatch.setattr(server_main, "bootstrap_services", lambda sess: finance_service)

    # Run the listener
    await server_main.run_booking_event_listener()

    # Verify RefundFailed event is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "RefundFailed"
    assert mock_publisher.events[0].booking_id == booking_id
    assert mock_publisher.events[0].client_id == client_id


async def test_check_balance_flow(finance_service):
    user_id = "user-cb-1"

    # 1. New user has no wallet pre-existing, check_balance should lazy-onboard and return False for positive amount
    has_sufficient = await finance_service.check_balance(user_id, 100)
    assert has_sufficient is False

    # Wallet should now exist with 0 balance
    wallet = await finance_service.wallet_repo.find_by_user_id(user_id)
    assert wallet is not None
    assert wallet.available_balance.amount == 0

    # Check balance for 0 amount should be True
    assert await finance_service.check_balance(user_id, 0) is True

    # 2. Deposit some money and check balance
    wallet.available_balance = Money(150)
    await finance_service.wallet_repo.save(wallet)

    assert await finance_service.check_balance(user_id, 100) is True
    assert await finance_service.check_balance(user_id, 150) is True
    assert await finance_service.check_balance(user_id, 200) is False
