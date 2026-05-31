import pytest
from unittest.mock import MagicMock
import uuid
import grpc

from internal.interfaces.grpc.servicer import DisputeServiceServicer
from gen.dispute.v1.messages.create_report_request_pb2 import (
    CreateReportRequest,
    EvidenceItem,
)
from gen.dispute.v1.messages.resolve_dispute_request_pb2 import ResolveDisputeRequest
from internal.infrastructure.persistence.models import DisputeModel
from sqlalchemy import select

pytestmark = pytest.mark.asyncio


@pytest.fixture
def grpc_servicer(test_session_factory):
    return DisputeServiceServicer(test_session_factory)


async def test_grpc_create_report_success(grpc_servicer, db_session):
    # Mock context with Istio auth headers
    context = MagicMock()
    context.invocation_metadata.return_value = [
        ("user-id", "client-id-111"),
        ("user-role", "CLIENT"),
    ]

    booking_id = str(uuid.uuid4())
    request = CreateReportRequest(
        booking_id=booking_id,
        reporter_id="client-id-111",
        accused_id="companion-id-222",
        reason="NO_SHOW",
        evidences=[EvidenceItem(type="TEXT", content="No show companion")],
    )

    response = await grpc_servicer.CreateReport(request, context)
    assert response.status == "SUCCESS"
    assert response.dispute_id != ""

    # Verify dispute in database
    dispute_db = (
        await db_session.execute(
            select(DisputeModel).filter_by(dispute_id=response.dispute_id)
        )
    ).scalar_one()
    assert dispute_db.booking_id == booking_id
    assert dispute_db.status == "OPEN"


async def test_grpc_resolve_dispute_success(
    grpc_servicer, db_session, integration_deps
):
    cmd_service = integration_deps["cmd_service"]

    # 1. Create a dispute first and assign admin
    booking_id = str(uuid.uuid4())
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-id-111",
        accused_id="companion-id-222",
        reason="NO_SHOW",
    )
    await cmd_service.assign_admin(dispute_id, "admin-id-789")
    await db_session.commit()

    # 2. Invoke ResolveDispute via gRPC
    context = MagicMock()
    context.invocation_metadata.return_value = [
        ("user-id", "admin-id-789"),
        ("user-role", "ADMIN"),
    ]

    request = ResolveDisputeRequest(
        dispute_id=dispute_id,
        admin_id="admin-id-789",
        resolution="REFUND_CLIENT",
        notes="Refunding because client requested it and it is valid.",
    )

    response = await grpc_servicer.ResolveDispute(request, context)
    assert response.status == "SUCCESS"
    assert response.dispute_id == dispute_id

    # Verify status changed in DB
    # Create fresh session to avoid cache issues
    async with db_session.bind.connect() as conn:
        result = await conn.execute(
            select(DisputeModel).filter_by(dispute_id=dispute_id)
        )
        dispute = result.first()
        assert dispute is not None
        assert dispute.status == "REFUNDED"
        assert dispute.resolution == "REFUND_CLIENT"


async def test_grpc_resolve_dispute_permission_denied(
    grpc_servicer, db_session, integration_deps
):
    cmd_service = integration_deps["cmd_service"]

    booking_id = str(uuid.uuid4())
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-id-111",
        accused_id="companion-id-222",
        reason="NO_SHOW",
    )
    await db_session.commit()

    # Context has CLIENT role instead of ADMIN
    context = MagicMock()
    context.invocation_metadata.return_value = [
        ("user-id", "client-id-111"),
        ("user-role", "CLIENT"),
    ]

    request = ResolveDisputeRequest(
        dispute_id=dispute_id,
        admin_id="client-id-111",
        resolution="REFUND_CLIENT",
        notes="Hacking attempt",
    )

    response = await grpc_servicer.ResolveDispute(request, context)
    # The default value for proto message status when uninitialized is empty string
    assert response.status == ""
    context.set_code.assert_called_with(grpc.StatusCode.PERMISSION_DENIED)
