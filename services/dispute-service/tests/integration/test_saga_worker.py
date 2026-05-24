import pytest
import uuid
from sqlalchemy import select
from internal.infrastructure.persistence.models import SagaStateModel
from internal.infrastructure.broker.saga_worker import SagaRetryWorker

pytestmark = pytest.mark.asyncio


async def test_saga_retry_worker_success(db_session, test_session_factory):
    """
    Test that SagaRetryWorker queries pending retriable sagas,
    processes them successfully using mock ports,
    and correctly commits changes to the database.
    """
    # 1. Arrange: Insert a pending, failed refund SAGA state
    saga_id = str(uuid.uuid4())
    dispute_id = str(uuid.uuid4())
    booking_id = str(uuid.uuid4())

    failed_saga = SagaStateModel(
        saga_id=saga_id,
        dispute_id=dispute_id,
        booking_id=booking_id,
        saga_type="REFUND",
        current_state="HIDING_REVIEW",  # A retriable state
        retry_count=1,
        last_error="Network timeout on interaction port",  # Makes it 'pending retry'
        version=1,
    )
    db_session.add(failed_saga)
    await db_session.commit()

    # Verify initially saved state in database
    init_db_state = (await db_session.execute(
        select(SagaStateModel).filter_by(saga_id=saga_id)
    )).scalar_one()
    assert init_db_state.current_state == "HIDING_REVIEW"
    assert init_db_state.last_error == "Network timeout on interaction port"

    # 2. Act: Initialize SagaRetryWorker and run the retry cycle manually
    worker = SagaRetryWorker(
        session_factory=test_session_factory,
        polling_interval_seconds=1.0
    )
    
    # Run a single loop iteration of retrying sagas
    await worker._retry_pending_sagas()

    # 3. Assert: Verify the SAGA state advanced to completed and last_error is cleared
    # Query database using a fresh session to ensure changes persisted
    async with test_session_factory() as verify_session:
        updated_db_state = (await verify_session.execute(
            select(SagaStateModel).filter_by(saga_id=saga_id)
        )).scalar_one()

        # Since MockInteractionAdapter succeeds by default, the retry will succeed.
        # It should advance state to DISPUTE_RESOLVED_REFUNDED and clear last_error.
        assert updated_db_state.current_state == "DISPUTE_RESOLVED_REFUNDED"
        assert updated_db_state.last_error is None
        assert updated_db_state.retry_count == 1
