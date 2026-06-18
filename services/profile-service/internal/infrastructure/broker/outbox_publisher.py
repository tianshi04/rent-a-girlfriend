import asyncio
import json
import logging
from datetime import datetime, timezone
from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker
from sqlalchemy import select
from aiokafka import AIOKafkaProducer
from cloudevents.v1.http import CloudEvent
from cloudevents.v1.conversion import to_structured
from internal.domain.events import DomainEvent
from internal.application.port import IEventPublisher
from internal.infrastructure.persistence.models import OutboxModel

logger = logging.getLogger("outbox_publisher")


class DatabaseEventPublisher(IEventPublisher):
    """
    Saves events directly to the Outbox table in the database.
    Part of the atomic business transaction.
    """

    def __init__(self, session: AsyncSession):
        self.session = session

    def publish(self, event: DomainEvent) -> None:
        import uuid
        from google.protobuf.json_format import MessageToDict
        from internal.infrastructure.mappers.event_mapper import EventMapper

        # Convert pure domain event to generated Protobuf message
        proto_msg = EventMapper.to_protobuf(event)

        # Strictly generate JSON payload based on proto contract
        payload_dict = MessageToDict(
            proto_msg, preserving_proto_field_name=False, use_integers_for_enums=True
        )

        event_id = str(uuid.uuid4())

        import re

        # Convert PascalCase/CamelCase to kebab-case
        # e.g., ProfileCreated -> profile-created
        name = proto_msg.DESCRIPTOR.name
        s1 = re.sub("(.)([A-Z][a-z]+)", r"\1-\2", name)
        kebab_name = re.sub("([a-z0-9])([A-Z])", r"\1-\2", s1).lower()

        outbox = OutboxModel(
            event_id=event_id,
            event_type=f"profile.{kebab_name}.v1",
            payload=json.dumps(payload_dict),
        )
        self.session.add(outbox)


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

                    # Construct standard CloudEvent v1.0 payload using the official SDK
                    time_str = (
                        event.created_at.replace(tzinfo=timezone.utc).isoformat()
                        if hasattr(event, "created_at") and event.created_at
                        else datetime.now(timezone.utc).isoformat()
                    )

                    attributes = {
                        "type": event.event_type,
                        "source": f"/rent-a-gf/profile-service/{payload.get('companionId') or payload.get('userId')}",
                        "id": event.event_id,
                        "time": time_str,
                        "datacontenttype": "application/json",
                        "correlationid": payload.get("eventId", event.event_id),
                    }

                    cloudevent = CloudEvent(attributes, payload)
                    _, body = to_structured(cloudevent)
                    cloudevent_dict = json.loads(body.decode("utf-8"))

                    # Publish to Kafka
                    if self.producer:
                        await self.producer.send_and_wait(
                            topic=self.topic,
                            key=bytes(payload.get("companionId", ""), "utf-8"),
                            value=cloudevent_dict,
                        )

                    # Mark as processed
                    event.processed = True

                await session.commit()
                logger.info("Outbox batch successfully committed and published")
            except Exception as e:
                await session.rollback()
                raise e
