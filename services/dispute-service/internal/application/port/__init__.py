from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional
from internal.domain.events import DomainEvent


@dataclass
class RefundResult:
    success: bool
    transaction_id: Optional[str] = None
    error: Optional[str] = None


@dataclass
class PayoutResult:
    success: bool
    transaction_id: Optional[str] = None
    error: Optional[str] = None


class IEventPublisher(ABC):
    @abstractmethod
    def publish(self, event: DomainEvent) -> None:
        """
        Publish event to broker. Under hexagonal, this is handled
        via Transactional Outbox pattern.
        """
        pass


class IFinancePort(ABC):
    @abstractmethod
    async def refund_escrow_to_wallet(self, booking_id: str) -> RefundResult:
        """gRPC client call to Finance Service to refund money from escrow."""
        pass

    @abstractmethod
    async def payout_from_escrow(
        self, booking_id: str, companion_wallet_id: str, commission_rate: float
    ) -> PayoutResult:
        """gRPC client call to Finance Service to payout from escrow."""
        pass

    @abstractmethod
    async def get_payout_snapshot(self, booking_id: str) -> tuple[str, float]:
        """Fetch snapshot details (companion_wallet_id, commission_rate) from Finance Service."""
        pass


class IInteractionPort(ABC):
    @abstractmethod
    async def hide_review_and_lock_chat(self, booking_id: str) -> bool:
        """gRPC client call to Interaction Service to hide client review and lock chat."""
        pass

    @abstractmethod
    async def lock_chat_room(self, booking_id: str) -> bool:
        """gRPC client call to Interaction Service to lock chat room."""
        pass
