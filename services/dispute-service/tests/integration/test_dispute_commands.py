import pytest
import uuid
from internal.domain.errors import DuplicateOpenDisputeError
from internal.infrastructure.persistence.models import (
    DisputeModel,
    DisputeEvidenceModel,
    OutboxModel,
    SagaStateModel,
)
from sqlalchemy import select

pytestmark = pytest.mark.asyncio


async def test_create_report_success(integration_deps):
    cmd_service = integration_deps["cmd_service"]
    session = integration_deps["session"]

    booking_id = str(uuid.uuid4())
    reporter_id = "client-123"
    accused_id = "companion-456"
    reason = "NO_SHOW"
    evidences = [
        {"evidence_type": "TEXT", "content": "Companion did not show up at the venue."},
        {"evidence_type": "IMAGE", "content": "https://example.com/evidence.jpg"},
    ]

    # Create report
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id=reporter_id,
        accused_id=accused_id,
        reason=reason,
        evidences=evidences,
    )

    assert dispute_id is not None
    await session.commit()

    # Query directly from database to verify persistence
    dispute_result = await session.execute(
        select(DisputeModel).filter_by(dispute_id=dispute_id)
    )
    dispute_db = dispute_result.scalar_one_or_none()
    assert dispute_db is not None
    assert dispute_db.booking_id == booking_id
    assert dispute_db.reporter_id == reporter_id
    assert dispute_db.accused_id == accused_id
    assert dispute_db.reason == reason
    assert dispute_db.status == "OPEN"
    assert dispute_db.admin_id is None

    # Check evidences
    evidence_result = await session.execute(
        select(DisputeEvidenceModel).filter_by(dispute_id=dispute_id)
    )
    evidences_db = evidence_result.scalars().all()
    assert len(evidences_db) == 2
    types = {ev.evidence_type for ev in evidences_db}
    assert "TEXT" in types
    assert "IMAGE" in types

    # Check that events were published to the outbox (DisputeCreated.v1)
    outbox_result = await session.execute(
        select(OutboxModel).order_by(OutboxModel.id.desc())
    )
    outbox_events = outbox_result.scalars().all()
    assert len(outbox_events) >= 1
    assert any("dispute-created" in e.event_type for e in outbox_events)


async def test_create_report_duplicate(integration_deps):
    cmd_service = integration_deps["cmd_service"]
    session = integration_deps["session"]

    booking_id = str(uuid.uuid4())

    # First report
    await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="NO_SHOW",
    )
    await session.commit()

    # Trying to create second open report for same booking should raise DuplicateOpenDisputeError
    with pytest.raises(DuplicateOpenDisputeError) as exc_info:
        await cmd_service.create_report(
            booking_id=booking_id,
            reporter_id="client-2",
            accused_id="companion-1",
            reason="OTHER",
        )
    assert booking_id in str(exc_info.value)


async def test_assign_admin_success(integration_deps):
    cmd_service = integration_deps["cmd_service"]
    session = integration_deps["session"]

    booking_id = str(uuid.uuid4())

    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="NO_SHOW",
    )
    await session.commit()

    # Assign Admin
    admin_id = "admin-999"
    await cmd_service.assign_admin(dispute_id, admin_id)
    await session.commit()

    # Verify status changed to RESOLVING
    dispute_result = await session.execute(
        select(DisputeModel).filter_by(dispute_id=dispute_id)
    )
    dispute_db = dispute_result.scalar_one()
    assert dispute_db.status == "RESOLVING"
    assert dispute_db.admin_id == admin_id


async def test_resolve_dispute_reject(integration_deps):
    cmd_service = integration_deps["cmd_service"]
    session = integration_deps["session"]

    booking_id = str(uuid.uuid4())
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="NO_SHOW",
    )
    await session.commit()

    admin_id = "admin-999"
    await cmd_service.assign_admin(dispute_id, admin_id)
    await session.commit()

    # Resolve with REJECT
    notes = "The companion provided solid evidence that they attended the meeting."
    await cmd_service.resolve_dispute(
        dispute_id=dispute_id,
        admin_id=admin_id,
        resolution="REJECT",
        notes=notes,
    )
    await session.commit()

    # Verify Dispute Model
    dispute_db = (
        await session.execute(select(DisputeModel).filter_by(dispute_id=dispute_id))
    ).scalar_one()
    assert dispute_db.status == "REJECTED"
    assert dispute_db.resolution == "REJECT"
    assert dispute_db.notes == notes

    # Verify Outbox contains DisputeResolved event and correct payload
    outbox_events = (await session.execute(select(OutboxModel))).scalars().all()
    resolved_event = next(
        e for e in outbox_events if "dispute-resolved" in e.event_type
    )
    import json

    payload = json.loads(resolved_event.payload)
    assert payload["bookingId"] == booking_id
    assert payload["resolvedBy"] == admin_id
    assert payload["reporterId"] == "client-1"
    assert payload["accusedId"] == "companion-1"


async def test_resolve_dispute_refund_saga(integration_deps):
    cmd_service = integration_deps["cmd_service"]
    session = integration_deps["session"]

    booking_id = str(uuid.uuid4())
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="NO_SHOW",
    )
    await session.commit()

    admin_id = "admin-999"
    await cmd_service.assign_admin(dispute_id, admin_id)
    await session.commit()

    # Resolve with REFUND_CLIENT -> Starts SAGA
    notes = "Refunding the client due to no-show."
    await cmd_service.resolve_dispute(
        dispute_id=dispute_id,
        admin_id=admin_id,
        resolution="REFUND_CLIENT",
        notes=notes,
    )
    await session.commit()

    # Verify Dispute Model status is REFUNDED
    dispute_db = (
        await session.execute(select(DisputeModel).filter_by(dispute_id=dispute_id))
    ).scalar_one()
    assert dispute_db.status == "REFUNDED"
    assert dispute_db.resolution == "REFUND_CLIENT"

    # Verify Saga state is DISPUTE_RESOLVED_REFUNDED
    saga_result = await session.execute(
        select(SagaStateModel).filter_by(dispute_id=dispute_id)
    )
    saga_db = saga_result.scalar_one_or_none()
    assert saga_db is not None
    assert saga_db.saga_type == "REFUND"
    assert saga_db.current_state == "DISPUTE_RESOLVED_REFUNDED"


async def test_resolve_dispute_payout_saga(integration_deps):
    cmd_service = integration_deps["cmd_service"]
    session = integration_deps["session"]

    booking_id = str(uuid.uuid4())
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="MISCONDUCT",
    )
    await session.commit()

    admin_id = "admin-999"
    await cmd_service.assign_admin(dispute_id, admin_id)
    await session.commit()

    # Resolve with PAYOUT_COMPANION -> Starts Payout SAGA
    notes = "Paying out to companion since service was rendered fully."
    await cmd_service.resolve_dispute(
        dispute_id=dispute_id,
        admin_id=admin_id,
        resolution="PAYOUT_COMPANION",
        notes=notes,
    )
    await session.commit()

    # Verify Dispute Model status is PAID_OUT
    dispute_db = (
        await session.execute(select(DisputeModel).filter_by(dispute_id=dispute_id))
    ).scalar_one()
    assert dispute_db.status == "PAID_OUT"
    assert dispute_db.resolution == "PAYOUT_COMPANION"

    # Verify Saga state is DISPUTE_RESOLVED_PAID_OUT
    saga_result = await session.execute(
        select(SagaStateModel).filter_by(dispute_id=dispute_id)
    )
    saga_db = saga_result.scalar_one_or_none()
    assert saga_db is not None
    assert saga_db.saga_type == "PAYOUT"
    assert saga_db.current_state == "DISPUTE_RESOLVED_PAID_OUT"

    # Verify snapshot fields are persisted
    assert saga_db.companion_wallet_id is not None
    assert saga_db.companion_wallet_id.startswith("wallet-")
    assert saga_db.commission_rate == 0.15
