import pytest
import json
from unittest.mock import AsyncMock
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker
from internal.bootstrap import Base
from internal.infrastructure.persistence.models import OutboxModel
from internal.infrastructure.broker.outbox_publisher import OutboxPublisherWorker

pytestmark = pytest.mark.asyncio


async def test_outbox_worker_publishes_valid_cloudevent():
    # 1. Setup sqlite in-memory database
    engine = create_async_engine("sqlite+aiosqlite:///:memory:", echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    SessionLocal = async_sessionmaker(
        autocommit=False, autoflush=False, bind=engine, class_=AsyncSession
    )

    # 2. Add an unprocessed event to outbox
    async with SessionLocal() as session:
        event = OutboxModel(
            event_id="test-event-uuid-123",
            event_type="profile.profile-created.v1",
            payload=json.dumps(
                {
                    "companionId": "comp-123",
                    "userId": "user-456",
                    "eventId": "corr-id-999",
                }
            ),
            processed=False,
        )
        session.add(event)
        await session.commit()

    # 3. Instantiate OutboxPublisherWorker with a mocked producer
    mock_producer = AsyncMock()

    worker = OutboxPublisherWorker(
        session_factory=SessionLocal,
        kafka_brokers="localhost:9092",
        topic="profile.events",
    )
    worker.producer = mock_producer

    # 4. Process a batch
    await worker._process_batch()

    # 5. Assertions
    # Verify the event was marked processed in the DB
    async with SessionLocal() as session:
        from sqlalchemy import select

        db_event = (
            await session.execute(
                select(OutboxModel).filter_by(event_id="test-event-uuid-123")
            )
        ).scalar_one()
        assert db_event.processed is True

    # Verify standard CloudEvent JSON format was published to mock producer
    assert mock_producer.send_and_wait.called
    kwargs = mock_producer.send_and_wait.call_args.kwargs
    assert kwargs["topic"] == "profile.events"
    assert kwargs["key"] == b"comp-123"

    cloudevent_dict = kwargs["value"]
    assert cloudevent_dict["specversion"] == "1.0"
    assert cloudevent_dict["id"] == "test-event-uuid-123"
    assert cloudevent_dict["source"] == "/rent-a-gf/profile-service/comp-123"
    assert cloudevent_dict["type"] == "profile.profile-created.v1"
    assert cloudevent_dict["datacontenttype"] == "application/json"
    assert "time" in cloudevent_dict
    assert cloudevent_dict["correlationid"] == "corr-id-999"
    assert cloudevent_dict["data"] == {
        "companionId": "comp-123",
        "userId": "user-456",
        "eventId": "corr-id-999",
    }

    await engine.dispose()


async def test_outbox_worker_correlationid_fallback():
    # 1. Setup sqlite in-memory database
    engine = create_async_engine("sqlite+aiosqlite:///:memory:", echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    SessionLocal = async_sessionmaker(
        autocommit=False, autoflush=False, bind=engine, class_=AsyncSession
    )

    # 2. Add an unprocessed event to outbox without eventId in payload
    async with SessionLocal() as session:
        event = OutboxModel(
            event_id="test-event-uuid-123",
            event_type="profile.profile-created.v1",
            payload=json.dumps({"companionId": "comp-123", "userId": "user-456"}),
            processed=False,
        )
        session.add(event)
        await session.commit()

    # 3. Instantiate OutboxPublisherWorker with a mocked producer
    mock_producer = AsyncMock()

    worker = OutboxPublisherWorker(
        session_factory=SessionLocal,
        kafka_brokers="localhost:9092",
        topic="profile.events",
    )
    worker.producer = mock_producer

    # 4. Process a batch
    await worker._process_batch()

    # 5. Assertions
    cloudevent_dict = mock_producer.send_and_wait.call_args.kwargs["value"]
    assert cloudevent_dict["correlationid"] == "test-event-uuid-123"

    await engine.dispose()
