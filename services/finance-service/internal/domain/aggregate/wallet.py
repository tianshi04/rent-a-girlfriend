from typing import List
from internal.domain.vo import Money
from internal.domain.errors import (
    InsufficientBalanceError,
    InsufficientFrozenBalanceError,
)
from internal.domain.events import (
    DomainEvent,
    CoinsFrozen,
    CoinsUnfrozen,
    WalletToppedUp,
)


class Wallet:
    def __init__(
        self,
        wallet_id: str,
        user_id: str,
        available_balance: Money,
        frozen_balance: Money,
    ):
        self.wallet_id = wallet_id
        self.user_id = user_id
        self.available_balance = available_balance
        self.frozen_balance = frozen_balance
        self.events: List[DomainEvent] = []

    def add_event(self, event: DomainEvent):
        self.events.append(event)

    def clear_events(self) -> List[DomainEvent]:
        events = self.events
        self.events = []
        return events

    @classmethod
    def create(cls, wallet_id: str, user_id: str) -> "Wallet":
        # New wallets start with 0 balance (INV-F01 is satisfied: available >= 0)
        return cls(
            wallet_id=wallet_id,
            user_id=user_id,
            available_balance=Money.zero(),
            frozen_balance=Money.zero(),
        )

    def freeze_coin(self, amount: Money, booking_id: str) -> None:
        """
        [INV-F02] AvailableBalance must be greater than or equal to the amount to freeze.
        """
        if self.available_balance.amount < amount.amount:
            raise InsufficientBalanceError(
                self.wallet_id, amount.amount, self.available_balance.amount
            )

        self.available_balance = self.available_balance.subtract(amount)
        self.frozen_balance = self.frozen_balance.add(amount)

        self.add_event(
            CoinsFrozen(
                wallet_id=self.wallet_id,
                user_id=self.user_id,
                amount=amount.amount,
                booking_id=booking_id,
            )
        )

    def unfreeze_coin(self, amount: Money, booking_id: str) -> None:
        """
        [INV-F03] Amount to unfreeze must not exceed current FrozenBalance.
        """
        if self.frozen_balance.amount < amount.amount:
            raise InsufficientFrozenBalanceError(
                self.wallet_id, amount.amount, self.frozen_balance.amount
            )

        self.frozen_balance = self.frozen_balance.subtract(amount)
        self.available_balance = self.available_balance.add(amount)

        self.add_event(
            CoinsUnfrozen(
                wallet_id=self.wallet_id,
                user_id=self.user_id,
                amount=amount.amount,
                booking_id=booking_id,
            )
        )

    def deduct_frozen(self, amount: Money) -> None:
        """
        Deducts coin from frozen balance (used when transferring coin to Escrow).
        [INV-F03] Deducted amount must not exceed current FrozenBalance.
        """
        if self.frozen_balance.amount < amount.amount:
            raise InsufficientFrozenBalanceError(
                self.wallet_id, amount.amount, self.frozen_balance.amount
            )

        self.frozen_balance = self.frozen_balance.subtract(amount)

    def topup(self, amount: Money, transaction_id: str, vnpay_amount_vnd: int) -> None:
        """
        Top up the wallet with Kano-Coins using external VNPay payment.
        """
        self.available_balance = self.available_balance.add(amount)
        self.add_event(
            WalletToppedUp(
                wallet_id=self.wallet_id,
                user_id=self.user_id,
                amount=amount.amount,
                vnpay_amount_vnd=vnpay_amount_vnd,
            )
        )

    def deposit(self, amount: Money) -> None:
        """
        General deposit (e.g. payout to companion or refund to client).
        """
        self.available_balance = self.available_balance.add(amount)
