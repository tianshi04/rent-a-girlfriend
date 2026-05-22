import asyncio
import logging
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker
from internal.domain.aggregate import DisputeRefundSaga, DisputePayoutSaga
from internal.infrastructure.persistence.repositories import SagaStateRepository
from internal.application.saga import DisputeRefundSagaOrchestrator, DisputePayoutSagaOrchestrator

logger = logging.getLogger("saga_retry_worker")


class SagaRetryWorker:
    """
    Background worker that polls failed SAGA states and retries the steps.
    Implements the Infinite Retry strategy for external integration steps (Interaction Service).
    """

    def __init__(
        self,
        session_factory: async_sessionmaker[AsyncSession],
        polling_interval_seconds: float = 5.0,
    ):
        self.session_factory = session_factory
        self.polling_interval = polling_interval_seconds
        self._running = False
        self._task: asyncio.Task | None = None

    async def start(self):
        logger.info("Starting Saga Retry Worker...")
        self._running = True
        self._task = asyncio.create_task(self._poll_loop())
        logger.info("Saga Retry Worker started successfully")

    async def stop(self):
        logger.info("Stopping Saga Retry Worker...")
        self._running = False
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        logger.info("Saga Retry Worker stopped")

    async def _poll_loop(self):
        while self._running:
            try:
                await self._retry_pending_sagas()
            except Exception as e:
                logger.error(f"Error in saga retry poll loop: {e}", exc_info=True)
            await asyncio.sleep(self.polling_interval)

    async def _retry_pending_sagas(self):
        from internal.bootstrap import bootstrap_services

        async with self.session_factory() as session:
            saga_repo = SagaStateRepository(session)
            pending = await saga_repo.find_pending_retries()
            if not pending:
                return

            logger.info(f"Saga Retry Worker found {len(pending)} pending sagas to retry")

            # Bootstrap command service for this specific transaction session
            cmd_service, _ = bootstrap_services(session)

            for saga in pending:
                try:
                    if isinstance(saga, DisputeRefundSaga):
                        # Re-process refund saga
                        logger.info(f"Retrying DisputeRefundSaga {saga.saga_id} (retry_count={saga.retry_count})")
                        await cmd_service.refund_saga_orchestrator.process_saga(saga)
                    elif isinstance(saga, DisputePayoutSaga):
                        # Re-process payout saga
                        logger.info(f"Retrying DisputePayoutSaga {saga.saga_id} (retry_count={saga.retry_count})")
                        await cmd_service.payout_saga_orchestrator.process_saga(saga)
                    
                    await session.commit()
                except Exception as e:
                    await session.rollback()
                    logger.error(f"Failed to retry saga {saga.saga_id}: {e}", exc_info=True)

