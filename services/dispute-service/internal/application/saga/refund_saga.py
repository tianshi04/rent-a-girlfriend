import logging
from internal.domain.aggregate import DisputeRefundSaga
from internal.domain.repository import ISagaStateRepository
from internal.application.port import IFinancePort, IInteractionPort, IEventPublisher

logger = logging.getLogger("refund_saga_orchestrator")


class DisputeRefundSagaOrchestrator:
    def __init__(
        self,
        saga_repo: ISagaStateRepository,
        finance_port: IFinancePort,
        interaction_port: IInteractionPort,
        event_publisher: IEventPublisher,
    ):
        self.saga_repo = saga_repo
        self.finance_port = finance_port
        self.interaction_port = interaction_port
        self.event_publisher = event_publisher

    async def start(self, saga_id: str, dispute_id: str, booking_id: str):
        logger.info(f"Starting DisputeRefundSaga {saga_id} for dispute {dispute_id}")
        saga = DisputeRefundSaga.create(saga_id, dispute_id, booking_id)
        await self.saga_repo.save(saga)
        await self.process_saga(saga)

    async def process_saga(self, saga: DisputeRefundSaga):
        logger.info(f"Processing DisputeRefundSaga {saga.saga_id} at state {saga.current_state}")
        
        if saga.current_state == "REFUNDING":
            try:
                res = await self.finance_port.refund_escrow_to_wallet(saga.booking_id)
                if res.success:
                    saga.on_refund_success()
                    await self.saga_repo.save(saga)
                    # Proceed to next step immediately
                    await self.process_saga(saga)
                else:
                    saga.on_refund_failed(res.error or "Refund failed")
                    await self.saga_repo.save(saga)
            except Exception as e:
                logger.error(f"Error in REFUNDING step of saga {saga.saga_id}: {e}")
                saga.on_refund_failed(str(e))
                await self.saga_repo.save(saga)

        elif saga.current_state == "HIDING_REVIEW":
            try:
                success = await self.interaction_port.hide_review_and_lock_chat(saga.booking_id)
                if success:
                    saga.on_hide_review_success()
                    await self.saga_repo.save(saga)
                else:
                    saga.on_hide_review_failed("Interaction service failed to lock/hide")
                    await self.saga_repo.save(saga)
            except Exception as e:
                logger.error(f"Error in HIDING_REVIEW step of saga {saga.saga_id}: {e}")
                saga.on_hide_review_failed(str(e))
                await self.saga_repo.save(saga)

        # Publish generated internal events
        for event in saga.clear_events():
            self.event_publisher.publish(event)
