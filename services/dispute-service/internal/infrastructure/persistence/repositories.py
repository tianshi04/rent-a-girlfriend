from typing import List, Optional, Tuple, Union
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func
from sqlalchemy.orm import selectinload

from internal.domain.aggregate import Dispute, DisputeRefundSaga, DisputePayoutSaga
from internal.domain.repository import IDisputeRepository, ISagaStateRepository
from internal.infrastructure.persistence.models import DisputeModel, SagaStateModel
from internal.infrastructure.mappers import DisputeMapper, SagaStateMapper


class DisputeRepository(IDisputeRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, dispute: Dispute) -> None:
        model = DisputeMapper.to_model(dispute)
        merged_model = await self.session.merge(model)
        await self.session.flush()
        # Sync auto-incremented OCC version back to the domain aggregate
        dispute.version = merged_model.version

    async def find_by_id(self, dispute_id: str) -> Optional[Dispute]:
        stmt = (
            select(DisputeModel)
            .options(selectinload(DisputeModel.evidences))
            .filter_by(dispute_id=dispute_id)
        )
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return DisputeMapper.to_domain(model)

    async def find_by_booking_id(self, booking_id: str) -> Optional[Dispute]:
        stmt = (
            select(DisputeModel)
            .options(selectinload(DisputeModel.evidences))
            .filter_by(booking_id=booking_id)
        )
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return DisputeMapper.to_domain(model)

    async def find_open_by_booking_id(self, booking_id: str) -> Optional[Dispute]:
        stmt = (
            select(DisputeModel)
            .options(selectinload(DisputeModel.evidences))
            .filter(
                DisputeModel.booking_id == booking_id,
                DisputeModel.status.in_(["OPEN", "RESOLVING"]),
            )
        )
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return DisputeMapper.to_domain(model)

    async def list_by_status(
        self, status: Optional[str], offset: int, limit: int
    ) -> Tuple[List[Dispute], int]:
        query = select(DisputeModel).options(selectinload(DisputeModel.evidences))
        if status:
            query = query.filter_by(status=status)

        # Count query
        count_query = select(func.count()).select_from(query.subquery())
        count_result = await self.session.execute(count_query)
        total = count_result.scalar() or 0

        # Pagination & sorting by creation time descending (or ID if no creation time available)
        query = (
            query.order_by(DisputeModel.created_at.desc()).offset(offset).limit(limit)
        )
        result = await self.session.execute(query)
        models = result.scalars().all()

        return [DisputeMapper.to_domain(m) for m in models], total


class SagaStateRepository(ISagaStateRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, saga: Union[DisputeRefundSaga, DisputePayoutSaga]) -> None:
        model = SagaStateMapper.to_model(saga)
        merged_model = await self.session.merge(model)
        await self.session.flush()
        # Sync auto-incremented OCC version back to the saga state
        saga.version = merged_model.version

    async def find_by_id(
        self, saga_id: str
    ) -> Optional[Union[DisputeRefundSaga, DisputePayoutSaga]]:
        stmt = select(SagaStateModel).filter_by(saga_id=saga_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return SagaStateMapper.to_domain(model)

    async def find_by_dispute_id(
        self, dispute_id: str
    ) -> Optional[Union[DisputeRefundSaga, DisputePayoutSaga]]:
        stmt = select(SagaStateModel).filter_by(dispute_id=dispute_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return SagaStateMapper.to_domain(model)

    async def find_pending_retries(
        self, limit: int = 20
    ) -> List[Union[DisputeRefundSaga, DisputePayoutSaga]]:
        """
        Find sagas that have a last_error and are in retriable states (HIDING_REVIEW or LOCKING_CHAT).
        """
        stmt = (
            select(SagaStateModel)
            .filter(
                SagaStateModel.last_error.isnot(None),
                SagaStateModel.current_state.in_(["HIDING_REVIEW", "LOCKING_CHAT"]),
            )
            .limit(limit)
        )
        result = await self.session.execute(stmt)
        models = result.scalars().all()
        return [SagaStateMapper.to_domain(m) for m in models]
