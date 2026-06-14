from dataclasses import dataclass


@dataclass(frozen=True, kw_only=True)
class DomainEvent:
    pass


@dataclass(frozen=True, kw_only=True)
class CoinsFrozen(DomainEvent):
    wallet_id: str
    user_id: str
    amount: int
    booking_id: str


@dataclass(frozen=True, kw_only=True)
class CoinsUnfrozen(DomainEvent):
    wallet_id: str
    user_id: str
    amount: int
    booking_id: str


@dataclass(frozen=True, kw_only=True)
class EscrowCreated(DomainEvent):
    booking_id: str
    amount: int


@dataclass(frozen=True, kw_only=True)
class PayoutProcessed(DomainEvent):
    booking_id: str
    companion_id: str
    amount: int
    commission_amount: int
    net_amount: int


@dataclass(frozen=True, kw_only=True)
class EscrowRefunded(DomainEvent):
    booking_id: str
    client_id: str
    refund_amount: int


@dataclass(frozen=True, kw_only=True)
class WalletToppedUp(DomainEvent):
    wallet_id: str
    user_id: str
    amount: int
    vnpay_amount_vnd: int


@dataclass(frozen=True, kw_only=True)
class EscrowFailed(DomainEvent):
    booking_id: str
    client_id: str
    reason: str


@dataclass(frozen=True, kw_only=True)
class RefundFailed(DomainEvent):
    booking_id: str
    client_id: str
    reason: str


@dataclass(frozen=True, kw_only=True)
class CoinsFreezeFailed(DomainEvent):
    booking_id: str
    user_id: str
    amount: int
    reason: str
