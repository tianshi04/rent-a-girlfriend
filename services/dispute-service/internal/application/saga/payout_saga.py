import logging
from internal.domain.aggregate import DisputePayoutSaga
from internal.domain.repository import ISagaStateRepository
from internal.application.port import IFinancePort, IInteractionPort, IEventPublisher

logger = logging.getLogger("payout_saga_orchestrator")


class DisputePayoutSagaOrchestrator:
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
        logger.info(f"Starting DisputePayoutSaga {saga_id} for dispute {dispute_id}")
        wallet_id, commission_rate = await self.finance_port.get_payout_snapshot(booking_id)
        saga = DisputePayoutSaga.create(
            saga_id=saga_id,
            dispute_id=dispute_id,
            booking_id=booking_id,
            companion_wallet_id=wallet_id,
            commission_rate=commission_rate,
        )
        await self.saga_repo.save(saga)
        await self.process_saga(saga)

    async def process_saga(self, saga: DisputePayoutSaga):
        logger.info(f"Processing DisputePayoutSaga {saga.saga_id} at state {saga.current_state}")
        
        if saga.current_state == "PAYING_OUT":
            try:
                # Use snapshot from saga state instead of hardcoding
                res = await self.finance_port.payout_from_escrow(
                    saga.booking_id, saga.companion_wallet_id, saga.commission_rate
                )
                if res.success:
                    saga.on_payout_success()
                    await self.saga_repo.save(saga)
                    # Proceed to next step immediately
                    await self.process_saga(saga)
                else:
                    saga.on_payout_failed(res.error or "Payout failed")
                    await self.saga_repo.save(saga)
            except Exception as e:
                logger.error(f"Error in PAYING_OUT step of saga {saga.saga_id}: {e}")
                saga.on_payout_failed(str(e))
                await self.saga_repo.save(saga)

        elif saga.current_state == "LOCKING_CHAT":
            try:
                success = await self.interaction_port.lock_chat_room(saga.booking_id)
                if success:
                    saga.on_lock_chat_success()
                    await self.saga_repo.save(saga)
                else:
                    saga.on_lock_chat_failed("Interaction service failed to lock chat room")
                    await self.saga_repo.save(saga)
            except Exception as e:
                logger.error(f"Error in LOCKING_CHAT step of saga {saga.saga_id}: {e}")
                saga.on_lock_chat_failed(str(e))
                await self.saga_repo.save(saga)

        # Publish generated internal events
        for event in saga.clear_events():
            self.event_publisher.publish(event)
