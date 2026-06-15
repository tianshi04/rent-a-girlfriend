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
