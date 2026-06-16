import asyncio
import json
import logging
from datetime import datetime, timezone
from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker
from sqlalchemy import select
from aiokafka import AIOKafkaProducer
from internal.infrastructure.persistence.models import OutboxModel
from cloudevents.v1.http import CloudEvent

logger = logging.getLogger("outbox_publisher")


class OutboxPublisherWorker:
    """
    Background worker that polls the Outbox table and publishes events to Kafka.
    Guarantees At-Least-Once Delivery.
    """

    def __init__(
        self,
        session_factory: async_sessionmaker[AsyncSession],
        kafka_brokers: str,
        topic: str,
        polling_interval_ms: int = 500,
        batch_size: int = 50,
    ):
        self.session_factory = session_factory
        self.kafka_brokers = kafka_brokers
        self.topic = topic
        self.polling_interval = polling_interval_ms / 1000.0
        self.batch_size = batch_size
        self.producer: Optional[AIOKafkaProducer] = None
        self._running = False
        self._task: Optional[asyncio.Task] = None

    async def start(self):
        logger.info("Starting Outbox Publisher Worker...")
        self.producer = AIOKafkaProducer(
            bootstrap_servers=self.kafka_brokers,
            value_serializer=lambda v: json.dumps(v).encode("utf-8"),
        )
        await self.producer.start()
        self._running = True
        self._task = asyncio.create_task(self._poll_loop())
        logger.info("Outbox Publisher Worker started successfully")

    async def stop(self):
        logger.info("Stopping Outbox Publisher Worker...")
        self._running = False
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        if self.producer:
            await self.producer.stop()
        logger.info("Outbox Publisher Worker stopped")

    async def _poll_loop(self):
        while self._running:
            try:
                await self._process_batch()
            except Exception as e:
                logger.error(f"Error in outbox poll loop: {e}", exc_info=True)
            await asyncio.sleep(self.polling_interval)

    async def _process_batch(self):
        async with self.session_factory() as session:
            try:
                # Query unprocessed events
                stmt = (
                    select(OutboxModel)
                    .filter(OutboxModel.processed.is_(False))
                    .order_by(OutboxModel.created_at.asc())
                    .limit(self.batch_size)
                )
                result = await session.execute(stmt)
                events = result.scalars().all()

                if not events:
                    return

                logger.info(f"Processing outbox batch: {len(events)} events")

                for event in events:
                    payload = json.loads(event.payload)

                    # Construct standard CloudEvent v1.0 payload using SDK
                    attributes = {
                        "type": event.event_type,
                        "source": f"/rent-a-gf/dispute-service/{payload.get('disputeId') or payload.get('bookingId') or 'system'}",
                        "correlationid": payload.get("correlationId") or payload.get("eventId") or event.event_id,
                        "datacontenttype": "application/json",
                        "id": event.event_id,
                    }
                    if hasattr(event, "created_at") and event.created_at:
                        attributes["time"] = event.created_at.isoformat()
                    else:
                        attributes["time"] = datetime.now(timezone.utc).isoformat()
                        
                    ce = CloudEvent(attributes, payload)
                    
                    # Convert CloudEvent to dict for JSON serialization
                    ce_dict = ce._attributes.copy()
                    ce_dict["data"] = ce.data

                    # Construct robust partition key
                    partition_key = payload.get("disputeId") or payload.get("bookingId") or payload.get("reporterId") or event.event_id

                    # Publish to Kafka
                    if self.producer:
                        await self.producer.send_and_wait(
                            topic=self.topic,
                            key=bytes(str(partition_key), "utf-8"),
                            value=ce_dict,
                        )

                    # Mark as processed
                    event.processed = True

                await session.commit()
                logger.info("Outbox batch successfully committed and published")
            except Exception as e:
                await session.rollback()
                raise e
