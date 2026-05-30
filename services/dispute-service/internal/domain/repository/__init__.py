from abc import ABC, abstractmethod
from typing import List, Optional, Union
from internal.domain.aggregate import Dispute, DisputeRefundSaga, DisputePayoutSaga


class IDisputeRepository(ABC):
    @abstractmethod
    async def save(self, dispute: Dispute) -> None:
        pass

    @abstractmethod
    async def find_by_id(self, dispute_id: str) -> Optional[Dispute]:
        pass

    @abstractmethod
    async def find_by_booking_id(self, booking_id: str) -> Optional[Dispute]:
        pass

    @abstractmethod
    async def find_open_by_booking_id(self, booking_id: str) -> Optional[Dispute]:
        """Find dispute with status OPEN or RESOLVING for a booking. Used for [INV-D01] check."""
        pass

    @abstractmethod
    async def list_by_status(
        self, status: Optional[str], offset: int, limit: int
    ) -> tuple[List[Dispute], int]:
        """List disputes with optional status filter, returns (disputes, total_count)."""
        pass


class ISagaStateRepository(ABC):
    @abstractmethod
    async def save(self, saga: Union[DisputeRefundSaga, DisputePayoutSaga]) -> None:
        pass

    @abstractmethod
    async def find_by_id(
        self, saga_id: str
    ) -> Optional[Union[DisputeRefundSaga, DisputePayoutSaga]]:
        pass

    @abstractmethod
    async def find_by_dispute_id(
        self, dispute_id: str
    ) -> Optional[Union[DisputeRefundSaga, DisputePayoutSaga]]:
        pass

    @abstractmethod
    async def find_pending_retries(
        self, limit: int = 20
    ) -> List[Union[DisputeRefundSaga, DisputePayoutSaga]]:
        """Find sagas that have a last_error and need retry (for Saga Retry Worker)."""
        pass
