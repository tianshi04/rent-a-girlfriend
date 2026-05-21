import asyncio
import json
import logging
from datetime import datetime, timezone
from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker
from sqlalchemy import select
from aiokafka import AIOKafkaProducer
from internal.domain.events import DomainEvent
from internal.application.port import IEventPublisher
from internal.infrastructure.persistence.models import OutboxModel

logger = logging.getLogger("outbox_publisher")


class DatabaseEventPublisher(IEventPublisher):
    """
    Saves events directly to the Outbox table in the database as part of active transaction.
    """

    def __init__(self, session: AsyncSession):
        self.session = session

    def publish(self, event: DomainEvent) -> None:
        import uuid
        from google.protobuf.json_format import MessageToDict
        from internal.infrastructure.mappers.event_mapper import EventMapper

        # Map to protobuf
        proto_msg = EventMapper.to_protobuf(event)

        # Convert to dictionary (preserving field names)
        payload_dict = MessageToDict(
            proto_msg, preserving_proto_field_name=True, use_integers_for_enums=True
        )

        event_id = str(uuid.uuid4())

        outbox = OutboxModel(
            event_id=event_id,
            event_type=f"com.rentagf.finance.{proto_msg.DESCRIPTOR.name}.v1",
            payload=json.dumps(payload_dict),
        )
        self.session.add(outbox)


class OutboxPublisherWorker:
    """
    Background worker polling outbox table and pushing to Kafka topic.
    Guarantees At-Least-Once delivery.
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

                    # Build Standard CloudEvent v1.0 payload
                    cloudevent = {
                        "specversion": "1.0",
                        "id": event.event_id,
                        "source": f"/rent-a-gf/finance-service/{payload.get('user_id') or payload.get('companion_id', 'system')}",
                        "type": event.event_type,
                        "datacontenttype": "application/json",
                        "time": event.created_at.isoformat() + "Z"
                        if hasattr(event, "created_at")
                        else datetime.now(timezone.utc).isoformat(),
                        "data": payload,
                        "extensions": {
                            "correlationId": payload.get("event_id", event.event_id)
                        },
                    }

                    if self.producer:
                        # Direct key partitioning by booking_id or user_id for sequence preservation
                        key_str = (
                            payload.get("booking_id") or payload.get("user_id") or ""
                        )
                        await self.producer.send_and_wait(
                            topic=self.topic,
                            key=bytes(key_str, "utf-8"),
                            value=cloudevent,
                        )

                    event.processed = True

                await session.commit()
                logger.info("Outbox batch successfully committed and published")
            except Exception as e:
                await session.rollback()
                raise e
