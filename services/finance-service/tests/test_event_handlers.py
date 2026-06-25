import pytest
import asyncio
from typing import List

from internal.domain.vo import Money, TransactionType
from internal.domain.events import DomainEvent
from internal.application.port import IEventPublisher
from internal.application.command.finance import FinanceCommandService
from internal.infrastructure.persistence.repositories import (
    WalletRepository,
    EscrowRepository,
    TransactionRepository,
)
from internal.infrastructure.payment.vnpay import VNPayAdapter
import main as server_main


class MockEventPublisher(IEventPublisher):
    def __init__(self):
        self.events: List[DomainEvent] = []

    def publish(self, event: DomainEvent) -> None:
        self.events.append(event)

    def clear(self):
        self.events = []


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
        session=db_session,
        vnpay_adapter=mock_vnpay_adapter,
    )


class MockMessage:
    def __init__(self, value):
        self.value = value


class MockConsumer:
    def __init__(self, messages):
        self.messages = messages

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


async def setup_mock_consumer(monkeypatch, db_session, finance_service, message_value):
    class MockSessionContext:
        def __init__(self, session):
            self.session = session

        async def __aenter__(self):
            return self.session

        async def __aexit__(self, exc_type, exc_val, exc_tb):
            pass

    monkeypatch.setattr(
        server_main,
        "AIOKafkaConsumer",
        lambda *args, **kwargs: MockConsumer([MockMessage(message_value)]),
    )
    monkeypatch.setattr(
        server_main, "SessionLocal", lambda: MockSessionContext(db_session)
    )
    monkeypatch.setattr(server_main, "bootstrap_services", lambda sess: finance_service)


async def test_booking_cancelled_early_unfreeze(
    finance_service, db_session, mock_publisher, monkeypatch
):
    """Test early cancellation unfreezes coins if no escrow exists."""
    client_id = "client-early-1"
    booking_id = "booking-early-1"

    # Set up wallet and freeze coins
    wallet = await finance_service.get_or_create_wallet(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(wallet)
    await db_session.commit()

    await finance_service.freeze_coin(
        user_id=client_id,
        amount=80,
        booking_id=booking_id,
    )
    await db_session.commit()

    mock_publisher.clear()

    # Early cancel event
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-111",
        "type": "booking.booking-cancelled-early.v1",
        "data": {
            "bookingId": booking_id,
            "clientId": client_id,
            "companionId": "comp-1",
        },
    }

    await setup_mock_consumer(monkeypatch, db_session, finance_service, mock_msg_value)
    await server_main.run_booking_event_listener()

    # Verify wallet balances (80 coins should be unfrozen, back to 100 available)
    db_session.expire_all()
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 100
    assert w.frozen_balance.amount == 0

    # Verify refund transaction
    txn = await finance_service.transaction_repo.find_by_reference_id(
        booking_id, TransactionType.REFUND
    )
    assert txn is not None
    assert txn.status == "SUCCESS"

    # Verify outbox event (CoinsUnfrozen) is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "CoinsUnfrozen"
    assert mock_publisher.events[0].amount == 80


async def test_booking_cancelled_early_refund_escrow(
    finance_service, db_session, mock_publisher, monkeypatch
):
    """Test early cancellation refunds escrow if escrow exists."""
    client_id = "client-early-2"
    booking_id = "booking-early-2"

    # Set up wallet and transfer to escrow
    wallet = await finance_service.get_or_create_wallet(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(wallet)
    await db_session.commit()

    await finance_service.freeze_coin(
        user_id=client_id,
        amount=80,
        booking_id=booking_id,
    )
    await db_session.commit()

    await finance_service.transfer_to_escrow(
        user_id=client_id,
        amount=80,
        booking_id=booking_id,
    )
    await db_session.commit()

    mock_publisher.clear()

    # Early cancel event
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-112",
        "type": "booking.booking-cancelled-early.v1",
        "data": {
            "bookingId": booking_id,
            "clientId": client_id,
            "companionId": "comp-1",
        },
    }

    await setup_mock_consumer(monkeypatch, db_session, finance_service, mock_msg_value)
    await server_main.run_booking_event_listener()

    # Verify escrow is refunded
    db_session.expire_all()
    escrow = await finance_service.escrow_repo.find_by_booking_id(booking_id)
    assert escrow.status == "REFUNDED"

    # Verify wallet balances (80 coins deposited back, back to 100 available)
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 100
    assert w.frozen_balance.amount == 0

    # Verify outbox event (EscrowRefunded) is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "EscrowRefunded"
    assert mock_publisher.events[0].refund_amount == 80


async def test_booking_cancelled_late_companion_refunds(
    finance_service, db_session, mock_publisher, monkeypatch
):
    """Test late cancellation by Companion refunds the client."""
    client_id = "client-late-1"
    booking_id = "booking-late-1"

    # Set up wallet and transfer to escrow
    wallet = await finance_service.get_or_create_wallet(client_id)
    wallet.available_balance = wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(wallet)
    await db_session.commit()

    await finance_service.freeze_coin(
        user_id=client_id,
        amount=80,
        booking_id=booking_id,
    )
    await db_session.commit()

    await finance_service.transfer_to_escrow(
        user_id=client_id,
        amount=80,
        booking_id=booking_id,
    )
    await db_session.commit()

    mock_publisher.clear()

    # Late cancel event by companion
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-113",
        "type": "booking.booking-cancelled-late.v1",
        "data": {
            "bookingId": booking_id,
            "clientId": client_id,
            "companionId": "comp-1",
            "actorRole": "COMPANION",
        },
    }

    await setup_mock_consumer(monkeypatch, db_session, finance_service, mock_msg_value)
    await server_main.run_booking_event_listener()

    # Verify escrow is refunded
    db_session.expire_all()
    escrow = await finance_service.escrow_repo.find_by_booking_id(booking_id)
    assert escrow.status == "REFUNDED"

    # Verify client wallet balance
    w = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w.available_balance.amount == 100


async def test_booking_cancelled_late_client_payouts(
    finance_service, db_session, mock_publisher, monkeypatch
):
    """Test late cancellation by Client pays out to the companion wallet."""
    client_id = "client-late-2"
    booking_id = "booking-late-2"
    companion_id = "companion-late-2"

    # Set up wallets and transfer to escrow
    client_wallet = await finance_service.get_or_create_wallet(client_id)
    client_wallet.available_balance = client_wallet.available_balance.add(Money(100))
    await finance_service.wallet_repo.save(client_wallet)

    companion_wallet = await finance_service.get_or_create_wallet(companion_id)
    assert companion_wallet.available_balance.amount == 0

    await db_session.commit()

    await finance_service.freeze_coin(
        user_id=client_id,
        amount=80,
        booking_id=booking_id,
    )
    await db_session.commit()

    await finance_service.transfer_to_escrow(
        user_id=client_id,
        amount=80,
        booking_id=booking_id,
    )
    await db_session.commit()

    mock_publisher.clear()

    # Late cancel event by client
    mock_msg_value = {
        "specversion": "1.0",
        "id": "event-114",
        "type": "booking.booking-cancelled-late.v1",
        "data": {
            "bookingId": booking_id,
            "clientId": client_id,
            "companionId": companion_id,
            "actorRole": "CLIENT",
        },
    }

    await setup_mock_consumer(monkeypatch, db_session, finance_service, mock_msg_value)
    await server_main.run_booking_event_listener()

    # Verify escrow is paid out
    db_session.expire_all()
    escrow = await finance_service.escrow_repo.find_by_booking_id(booking_id)
    assert escrow.status == "PAID_OUT"

    # Verify companion wallet balance received 80 coins (0% commission)
    w_comp = await finance_service.wallet_repo.find_by_user_id(companion_id)
    assert w_comp.available_balance.amount == 80

    # Verify client wallet balance remains 20 (since cọc was deducted and transferred to companion)
    w_client = await finance_service.wallet_repo.find_by_user_id(client_id)
    assert w_client.available_balance.amount == 20

    # Verify outbox event (PayoutProcessed) is published
    assert len(mock_publisher.events) == 1
    assert mock_publisher.events[0].__class__.__name__ == "PayoutProcessed"
    assert mock_publisher.events[0].net_amount == 80
    assert mock_publisher.events[0].commission_amount == 0
