import pytest
from unittest.mock import MagicMock

from internal.interfaces.grpc.servicer import ProfileServiceServicer
from gen.profile.v1.messages.create_profile_request_pb2 import CreateProfileRequest
from gen.profile.v1.messages.approve_profile_request_pb2 import ApproveProfileRequest

pytestmark = pytest.mark.asyncio


@pytest.fixture
def grpc_servicer(TestSessionLocal):
    return ProfileServiceServicer(TestSessionLocal)


async def test_grpc_create_profile_success(grpc_servicer, db_session, integration_deps):
    # Mock gRPC Context
    context = MagicMock()
    # Mock Istio authenticated headers in context
    context.invocation_metadata.return_value = [
        ("user-id", "companion_user_456"),
        ("user-role", "COMPANION"),
    ]

    request = CreateProfileRequest(
        user_id="companion_user_456",
        display_name="Asami Mami",
        intro_text="Cute and lovely rental girlfriend",
        available_cities=["Hanoi", "HCM"],
    )

    response = await grpc_servicer.CreateProfile(request, context)
    assert response.status == "SUCCESS"
    assert response.companion_id == "companion_user_456"

    # Verify saved state in DB
    await db_session.commit()
    profile_repo = integration_deps["profile_repo"]
    profile = await profile_repo.find_by_id("companion_user_456")
    assert profile is not None
    assert profile.display_name == "Asami Mami"
    assert profile.status == "PENDING"  # Starts as pending for manual admin approval


async def test_grpc_admin_approve_profile(grpc_servicer, db_session, integration_deps):
    # Ensure profile exists before approval
    profile_repo = integration_deps["profile_repo"]
    profile_cmd = integration_deps["profile_cmd"]
    if not await profile_repo.find_by_id("companion_user_456"):
        await profile_cmd.create_profile(
            companion_id="companion_user_456",
            user_id="companion_user_456",
            display_name="Asami Mami",
            intro_text="Cute rental girlfriend",
            available_cities=["Hanoi"],
        )
        await db_session.commit()

    context = MagicMock()
    # Admin context
    context.invocation_metadata.return_value = [
        ("user-id", "admin_user_99"),
        ("user-role", "ADMIN"),
    ]

    request = ApproveProfileRequest(
        companion_id="companion_user_456", admin_id="admin_user_99"
    )

    response = await grpc_servicer.ApproveProfile(request, context)
    assert response.status == "SUCCESS"

    # Verify status transition
    await db_session.commit()
    profile = await profile_repo.find_by_id("companion_user_456")
    assert profile.status == "APPROVED"
