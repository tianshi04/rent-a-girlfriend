import pytest
from httpx import AsyncClient, ASGITransport
import uuid
from internal.bootstrap import app

pytestmark = pytest.mark.asyncio


@pytest.fixture
async def client():
    async with AsyncClient(
        transport=ASGITransport(app=app), base_url="http://testserver/api/v1"
    ) as ac:
        yield ac


async def test_health_check(client):
    response = await client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "HEALTHY"}


async def test_get_disputes_unauthorized(client):
    # Missing auth headers
    response = await client.get("/disputes")
    assert response.status_code == 401
    assert "User authentication missing" in response.json()["detail"]


async def test_get_disputes_forbidden(client):
    # Non-admin role
    headers = {"user-id": "client-1", "user-role": "CLIENT"}
    response = await client.get("/disputes", headers=headers)
    assert response.status_code == 403
    assert "Admin permission required" in response.json()["detail"]


async def test_list_disputes_success(client, db_session, integration_deps):
    cmd_service = integration_deps["cmd_service"]

    # Create some disputes
    booking_1 = str(uuid.uuid4())
    booking_2 = str(uuid.uuid4())

    await cmd_service.create_report(
        booking_id=booking_1,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="NO_SHOW",
    )
    await cmd_service.create_report(
        booking_id=booking_2,
        reporter_id="client-2",
        accused_id="companion-2",
        reason="OTHER",
    )
    await db_session.commit()

    headers = {"user-id": "admin-123", "user-role": "ADMIN"}
    response = await client.get("/disputes", headers=headers)
    assert response.status_code == 200

    data = response.json()
    assert "disputes" in data
    assert data["total"] >= 2
    assert len(data["disputes"]) >= 2

    # Test status filtering
    response_filter = await client.get("/disputes?status=OPEN", headers=headers)
    assert response_filter.status_code == 200
    assert all(d["status"] == "OPEN" for d in response_filter.json()["disputes"])


async def test_get_dispute_detail(client, db_session, integration_deps):
    cmd_service = integration_deps["cmd_service"]

    booking_id = str(uuid.uuid4())
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="NO_SHOW",
        evidences=[{"evidence_type": "TEXT", "content": "Proof detail"}],
    )
    await db_session.commit()

    headers = {"user-id": "admin-123", "user-role": "ADMIN"}

    # Valid ID
    response = await client.get(f"/disputes/{dispute_id}", headers=headers)
    assert response.status_code == 200
    data = response.json()
    assert data["disputeId"] == dispute_id
    assert data["bookingId"] == booking_id
    assert data["reason"] == "NO_SHOW"
    assert len(data["evidences"]) == 1
    assert data["evidences"][0]["content"] == "Proof detail"

    # Non-existent ID
    response_missing = await client.get(f"/disputes/{uuid.uuid4()}", headers=headers)
    assert response_missing.status_code == 404


async def test_get_saga_state(client, db_session, integration_deps):
    cmd_service = integration_deps["cmd_service"]

    booking_id = str(uuid.uuid4())
    dispute_id = await cmd_service.create_report(
        booking_id=booking_id,
        reporter_id="client-1",
        accused_id="companion-1",
        reason="NO_SHOW",
    )
    await cmd_service.assign_admin(dispute_id, "admin-123")
    await cmd_service.resolve_dispute(
        dispute_id=dispute_id,
        admin_id="admin-123",
        resolution="REFUND_CLIENT",
        notes="Refunding client",
    )
    await db_session.commit()

    headers = {"user-id": "admin-123", "user-role": "ADMIN"}

    response = await client.get(f"/disputes/{dispute_id}/saga", headers=headers)
    assert response.status_code == 200
    saga_data = response.json()
    assert saga_data is not None
    assert saga_data["disputeId"] == dispute_id
    assert saga_data["bookingId"] == booking_id
    assert saga_data["sagaType"] == "REFUND"
    assert saga_data["currentState"] == "DISPUTE_RESOLVED_REFUNDED"
