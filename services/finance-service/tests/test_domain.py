import pytest
from internal.domain.vo import Money
from internal.domain.aggregate.wallet import Wallet
from internal.domain.aggregate.escrow import Escrow
from internal.domain.errors import (
    InvalidAmountError,
    InsufficientBalanceError,
    InsufficientFrozenBalanceError,
    InvalidEscrowStatusTransitionError,
)
from internal.domain.service import CommissionCalculatorService, CurrencyExchangeService


def test_money_validation():
    # Test valid money creation
    m = Money(100)
    assert m.amount == 100

    # Test negative amount error
    with pytest.raises(InvalidAmountError):
        Money(-1)

    # Test zero creation
    zero = Money.zero()
    assert zero.amount == 0

    # Test add/subtract operations
    m1 = Money(50)
    m2 = Money(30)
    assert m1.add(m2).amount == 80
    assert m1.subtract(m2).amount == 20

    # Subtracting more than available results in InvalidAmountError due to negative money
    with pytest.raises(InvalidAmountError):
        m2.subtract(m1)


def test_wallet_invariants_and_events():
    # [INV-F01] Wallet creation starts with 0 balances (available >= 0)
    wallet = Wallet.create(wallet_id="w-123", user_id="u-123")
    assert wallet.wallet_id == "w-123"
    assert wallet.user_id == "u-123"
    assert wallet.available_balance.amount == 0
    assert wallet.frozen_balance.amount == 0

    # Topup wallet
    wallet.topup(Money(100), transaction_id="t-1", vnpay_amount_vnd=100000)
    assert wallet.available_balance.amount == 100

    events = wallet.clear_events()
    assert len(events) == 1
    assert events[0].__class__.__name__ == "WalletToppedUp"
    assert events[0].amount == 100
    assert events[0].vnpay_amount_vnd == 100000

    # [INV-F02] Freeze coins (Must have enough available balance)
    # Freeze 40 coins
    wallet.freeze_coin(Money(40), booking_id="b-1")
    assert wallet.available_balance.amount == 60
    assert wallet.frozen_balance.amount == 40

    events = wallet.clear_events()
    assert len(events) == 1
    assert events[0].__class__.__name__ == "CoinsFrozen"
    assert events[0].amount == 40
    assert events[0].booking_id == "b-1"

    # [INV-F02 Failure case] Freeze more than available balance
    with pytest.raises(InsufficientBalanceError):
        wallet.freeze_coin(Money(70), booking_id="b-2")

    # [INV-F03] Unfreeze coins (Must not exceed frozen balance)
    wallet.unfreeze_coin(Money(10), booking_id="b-1")
    assert wallet.available_balance.amount == 70
    assert wallet.frozen_balance.amount == 30

    events = wallet.clear_events()
    assert len(events) == 1
    assert events[0].__class__.__name__ == "CoinsUnfrozen"
    assert events[0].amount == 10

    # [INV-F03 Failure case] Unfreeze more than frozen balance
    with pytest.raises(InsufficientFrozenBalanceError):
        wallet.unfreeze_coin(Money(40), booking_id="b-1")

    # Deduct frozen balance for escrow transfer
    wallet.deduct_frozen(Money(30))
    assert wallet.frozen_balance.amount == 0

    with pytest.raises(InsufficientFrozenBalanceError):
        wallet.deduct_frozen(Money(5))


def test_escrow_invariants_and_commission():
    # [INV-F04] Initial state of Escrow is HELD
    escrow = Escrow.create(escrow_id="e-1", booking_id="b-1", amount=Money(105))
    assert escrow.status == "HELD"
    assert escrow.amount.amount == 105

    events = escrow.clear_events()
    assert len(events) == 1
    assert events[0].__class__.__name__ == "EscrowCreated"
    assert events[0].amount == 105

    # [INV-F05] Process payout (Only from HELD status)
    # Payout with 10% commission (commission_rate = 0.1)
    # 105 * 0.1 = 10.5. Rounding 10.5 using math round() should be 10 (Python round(10.5) is 10, round(11.5) is 12 due to Bankers rounding,
    # but round(10.5) is indeed 10 in Python 3. Let's assert based on python's native round() logic).
    commission_val = int(round(105 * 0.1))
    expected_net = 105 - commission_val

    commission, net_amount = escrow.payout(companion_id="comp-1", commission_rate=0.1)
    assert escrow.status == "PAID_OUT"
    assert commission == commission_val
    assert net_amount == expected_net

    events = escrow.clear_events()
    assert len(events) == 1
    assert events[0].__class__.__name__ == "PayoutProcessed"
    assert events[0].commission_amount == commission
    assert events[0].net_amount == net_amount

    # [INV-F05 Failure case] Cannot payout again
    with pytest.raises(InvalidEscrowStatusTransitionError):
        escrow.payout(companion_id="comp-1", commission_rate=0.1)

    # Test refund from non-HELD status
    with pytest.raises(InvalidEscrowStatusTransitionError):
        escrow.refund(client_id="client-1", refund_amount=Money(105))


def test_escrow_refund():
    # Test successful refund from HELD
    escrow = Escrow.create(escrow_id="e-2", booking_id="b-2", amount=Money(100))
    # Clear the initial EscrowCreated event
    escrow.clear_events()

    escrow.refund(client_id="client-2", refund_amount=Money(100))
    assert escrow.status == "REFUNDED"

    events = escrow.clear_events()
    assert len(events) == 1
    assert events[0].__class__.__name__ == "EscrowRefunded"
    assert events[0].refund_amount == 100


def test_domain_services():
    # Test CurrencyExchangeService
    assert CurrencyExchangeService.coin_to_vnd(Money(5)) == 5000
    assert CurrencyExchangeService.vnd_to_coin(5000).amount == 5
    assert CurrencyExchangeService.vnd_to_coin(5500).amount == 5  # Integer division

    # Test CommissionCalculatorService
    commission = CommissionCalculatorService.calculate_commission(Money(15), 0.1)
    assert commission == 2  # round(1.5) = 2

    commission2 = CommissionCalculatorService.calculate_commission(Money(25), 0.1)
    assert commission2 == 2  # round(2.5) = 2 (banker's rounding)
