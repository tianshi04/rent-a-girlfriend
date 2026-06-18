import asyncio
import logging
from datetime import datetime, timezone, timedelta
from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker
from sqlalchemy import delete
from internal.infrastructure.persistence.models import OutboxModel, ProcessedEventModel

logger = logging.getLogger("db_cleanup_worker")


class DbCleanupWorker:
    """
    Background worker that periodically deletes old records from
    the processed_events and outbox tables to prevent unbounded growth.
    """

    def __init__(
        self,
        session_factory: async_sessionmaker[AsyncSession],
        cleanup_interval_minutes: int = 30,
        processed_events_retention_days: int = 7,
        outbox_retention_days: int = 1,
    ):
        self.session_factory = session_factory
        self.cleanup_interval_seconds = cleanup_interval_minutes * 60
        self.processed_events_retention = timedelta(
            days=processed_events_retention_days
        )
        self.outbox_retention = timedelta(days=outbox_retention_days)
        self._running = False
        self._task: Optional[asyncio.Task] = None

    async def start(self):
        logger.info("Starting Database Cleanup Worker...")
        self._running = True
        self._task = asyncio.create_task(self._poll_loop())
        logger.info("Database Cleanup Worker started successfully")

    async def stop(self):
        logger.info("Stopping Database Cleanup Worker...")
        self._running = False
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        logger.info("Database Cleanup Worker stopped")

    async def _poll_loop(self):
        while self._running:
            try:
                await self._run_cleanup()
            except Exception as e:
                logger.error(f"Error in DB cleanup loop: {e}", exc_info=True)

            # Wait for the next interval or handle cancellation
            try:
                await asyncio.sleep(self.cleanup_interval_seconds)
            except asyncio.CancelledError:
                break

    async def _run_cleanup(self):
        now = datetime.now(timezone.utc).replace(tzinfo=None)
        cutoff_events = now - self.processed_events_retention
        cutoff_outbox = now - self.outbox_retention

        logger.info(
            f"Running DB cleanup. Processed events cutoff: {cutoff_events}, Outbox cutoff: {cutoff_outbox}"
        )

        async with self.session_factory() as session:
            try:
                # 1. Delete old processed_events
                stmt_events = delete(ProcessedEventModel).where(
                    ProcessedEventModel.processed_at < cutoff_events
                )
                res_events = await session.execute(stmt_events)

                # 2. Delete old outbox records that have been processed
                stmt_outbox = delete(OutboxModel).where(
                    OutboxModel.processed.is_(True),
                    OutboxModel.created_at < cutoff_outbox,
                )
                res_outbox = await session.execute(stmt_outbox)

                await session.commit()

                logger.info(
                    f"Cleaned up {res_events.rowcount} old processed_events and {res_outbox.rowcount} outbox records."
                )
            except Exception as e:
                await session.rollback()
                raise e
