import uuid
from typing import List, Optional, Dict, Any
from internal.domain.aggregate import (
    Dispute,
    DisputeEvidence,
)
from internal.domain.repository import IDisputeRepository, ISagaStateRepository
from internal.domain.errors import DuplicateOpenDisputeError, DisputeNotFoundError
from internal.domain.service import DisputeRoutingService
from internal.application.port import IEventPublisher


class DisputeCommandService:
    def __init__(
        self,
        dispute_repo: IDisputeRepository,
        saga_repo: ISagaStateRepository,
        event_publisher: IEventPublisher,
        refund_saga_orchestrator: Any,  # Avoid circular dependency, will be typed or resolved at bootstrap
        payout_saga_orchestrator: Any,  # Same as above
    ):
        self.dispute_repo = dispute_repo
        self.saga_repo = saga_repo
        self.event_publisher = event_publisher
        self.refund_saga_orchestrator = refund_saga_orchestrator
        self.payout_saga_orchestrator = payout_saga_orchestrator
        self.routing_service = DisputeRoutingService()

    async def create_report(
        self,
        booking_id: str,
        reporter_id: str,
        accused_id: str,
        reason: str,
        evidences: Optional[List[Dict[str, str]]] = None,
    ) -> str:
        # 1. Validate [INV-D01]: Check if there is already an active (OPEN/RESOLVING) dispute for this booking
        existing = await self.dispute_repo.find_open_by_booking_id(booking_id)
        if existing:
            raise DuplicateOpenDisputeError(booking_id)

        # Generate dispute ID
        dispute_id = str(uuid.uuid4())

        # Map evidences dicts to DisputeEvidence entities
        evidence_entities = []
        if evidences:
            for idx, ev in enumerate(evidences):
                evidence_entities.append(
                    DisputeEvidence(
                        evidence_id=str(uuid.uuid4()),
                        evidence_type=ev.get("evidence_type", "TEXT"),
                        content=ev.get("content", ""),
                    )
                )

        # 2. Create the aggregate using factory method
        dispute = Dispute.create_report(
            dispute_id=dispute_id,
            booking_id=booking_id,
            reporter_id=reporter_id,
            accused_id=accused_id,
            reason=reason,
            evidences=evidence_entities,
        )

        # 3. Save to repository
        await self.dispute_repo.save(dispute)

        # 4. Publish generated domain events
        for event in dispute.clear_events():
            self.event_publisher.publish(event)

        return dispute_id

    async def assign_admin(self, dispute_id: str, admin_id: str) -> None:
        """Assign an admin to handle a dispute."""
        dispute = await self.dispute_repo.find_by_id(dispute_id)
        if not dispute:
            raise DisputeNotFoundError(dispute_id)

        dispute.assign_admin(admin_id)
        await self.dispute_repo.save(dispute)

        for event in dispute.clear_events():
            self.event_publisher.publish(event)

    async def resolve_dispute(
        self, dispute_id: str, admin_id: str, resolution: str, notes: str = ""
    ) -> None:
        """
        Resolve the dispute and kick off SAGA if financial action is required.
        """
        dispute = await self.dispute_repo.find_by_id(dispute_id)
        if not dispute:
            raise DisputeNotFoundError(dispute_id)

        # Route the resolution to corresponding domain actions
        if resolution == "REFUND_CLIENT":
            dispute.resolve_refund(admin_id, notes)
            await self.dispute_repo.save(dispute)

            # Start DisputeRefundSaga
            saga_id = str(uuid.uuid4())
            await self.refund_saga_orchestrator.start(
                saga_id=saga_id,
                dispute_id=dispute_id,
                booking_id=dispute.booking_id,
            )

        elif resolution == "PAYOUT_COMPANION":
            dispute.resolve_payout(admin_id, notes)
            await self.dispute_repo.save(dispute)

            # Start DisputePayoutSaga
            saga_id = str(uuid.uuid4())
            await self.payout_saga_orchestrator.start(
                saga_id=saga_id,
                dispute_id=dispute_id,
                booking_id=dispute.booking_id,
            )

        elif resolution == "REJECT":
            dispute.reject(admin_id, notes)
            await self.dispute_repo.save(dispute)
            # REJECT has no SAGA, only publishes event

        else:
            from internal.domain.errors import InvalidResolutionError
            from internal.domain.vo.dispute_reason import VALID_RESOLUTIONS

            raise InvalidResolutionError(resolution, VALID_RESOLUTIONS)

        # Publish the domain events from Dispute
        for event in dispute.clear_events():
            self.event_publisher.publish(event)
