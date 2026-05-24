from typing import List, Optional, Tuple, Union
from internal.domain.aggregate import Dispute, DisputeRefundSaga, DisputePayoutSaga
from internal.domain.repository import IDisputeRepository, ISagaStateRepository
from internal.domain.errors import DisputeNotFoundError


class DisputeQueryService:
    def __init__(self, dispute_repo: IDisputeRepository, saga_repo: ISagaStateRepository):
        self.dispute_repo = dispute_repo
        self.saga_repo = saga_repo

    async def list_disputes(
        self, status: Optional[str] = None, offset: int = 0, limit: int = 10
    ) -> Tuple[List[Dispute], int]:
        """List disputes with optional status filter. Returns (disputes, total_count)."""
        return await self.dispute_repo.list_by_status(status, offset, limit)

    async def get_dispute_detail(self, dispute_id: str) -> Dispute:
        """Get detail of a single dispute."""
        dispute = await self.dispute_repo.find_by_id(dispute_id)
        if not dispute:
            raise DisputeNotFoundError(dispute_id)
        return dispute

    async def get_saga_state(
        self, dispute_id: str
    ) -> Optional[Union[DisputeRefundSaga, DisputePayoutSaga]]:
        """Get the current SAGA execution state associated with a dispute (for Admin Dashboard debugging)."""
        return await self.saga_repo.find_by_dispute_id(dispute_id)
