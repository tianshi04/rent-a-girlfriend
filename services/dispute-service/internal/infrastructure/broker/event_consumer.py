import asyncio
import json
import logging
from typing import List, Optional
from aiokafka import AIOKafkaConsumer
from sqlalchemy import select
from sqlalchemy.ext.asyncio import async_sessionmaker, AsyncSession
from internal.infrastructure.persistence.models import ProcessedEventModel

logger = logging.getLogger("event_consumer")


class DisputeEventConsumer:
    """
    Kafka consumer for reply/integration events.
    Includes idempotency guards using ProcessedEventModel.
    """

    def __init__(
        self,
        session_factory: async_sessionmaker[AsyncSession],
        kafka_brokers: str,
        topics: List[str],
    ):
        self.session_factory = session_factory
        self.kafka_brokers = kafka_brokers
        self.topics = topics
        self.consumer: Optional[AIOKafkaConsumer] = None
        self._running = False
        self._task: Optional[asyncio.Task] = None

    async def start(self):
        logger.info(f"Starting Dispute Event Consumer for topics {self.topics}...")
        self.consumer = AIOKafkaConsumer(
            *self.topics,
            bootstrap_servers=self.kafka_brokers,
            group_id="dispute-service-group",
            value_deserializer=lambda m: json.loads(m.decode("utf-8")),
            auto_offset_reset="earliest",
        )
        try:
            await self.consumer.start()
        except Exception as e:
            logger.warning(
                f"Could not connect/start Kafka consumer (brokers {self.kafka_brokers}): {e}. SAGA will still work via gRPC sync client."
            )
            return

        self._running = True
        self._task = asyncio.create_task(self._consume_loop())
        logger.info("Dispute Event Consumer started successfully")

    async def stop(self):
        logger.info("Stopping Dispute Event Consumer...")
        self._running = False
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        if self.consumer:
            await self.consumer.stop()
        logger.info("Dispute Event Consumer stopped")

    async def _consume_loop(self):
        while self._running:
            try:
                # Poll messages
                msg_set = await self.consumer.getmany(timeout_ms=1000)
                for tp, messages in msg_set.items():
                    for msg in messages:
                        await self._process_message(msg.value)
            except Exception as e:
                logger.error(f"Error in consumer loop: {e}", exc_info=True)
                await asyncio.sleep(1)

    async def _process_message(self, cloudevent: dict):
        event_id = cloudevent.get("id")
        event_type = cloudevent.get("type")
        if not event_id:
            return

        async with self.session_factory() as session:
            try:
                # 1. Idempotency Check: check if event has already been processed
                stmt = select(ProcessedEventModel).filter_by(event_id=event_id)
                result = await session.execute(stmt)
                existing = result.scalars().first()
                if existing:
                    logger.info(f"Event {event_id} already processed. Skipping.")
                    return

                logger.info(f"Processing event: {event_id} ({event_type})")

                # --- Handle different event types here ---
                # Under the current synchronous gRPC-based SAGA design,
                # we don't have async replies, but this is fully set up for expansion.

                # 2. Mark event as processed
                processed = ProcessedEventModel(event_id=event_id)
                session.add(processed)
                await session.commit()
            except Exception as e:
                await session.rollback()
                logger.error(f"Failed to process event {event_id}: {e}", exc_info=True)
                raise e
