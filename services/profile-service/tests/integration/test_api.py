import pytest
from httpx import AsyncClient, ASGITransport
from internal.bootstrap import app

pytestmark = pytest.mark.asyncio


@pytest.fixture
async def client():
    async with AsyncClient(
        transport=ASGITransport(app=app), base_url="http://testserver/api/v1"
    ) as ac:
        yield ac


@pytest.fixture
async def root_client():
    async with AsyncClient(
        transport=ASGITransport(app=app), base_url="http://testserver"
    ) as ac:
        yield ac


@pytest.fixture(autouse=True)
async def seed_test_data(integration_deps, db_session):
    profile_cmd = integration_deps["profile_cmd"]
    profile_repo = integration_deps["profile_repo"]

    if not await profile_repo.find_by_id("user_companion_123"):
        await profile_cmd.create_profile(
            companion_id="user_companion_123",
            user_id="user_companion_123",
            display_name="Kano Chizuru",
            bio="Perfect rental girlfriend",
            available_cities=["Hanoi"],
            role="COMPANION",
        )
        await db_session.commit()


async def test_health_check(root_client):
    response = await root_client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok", "service": "profile-service"}


async def test_presigned_url_valid_image(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "assetType": "IMAGE",
        "sizeBytes": 1 * 1024 * 1024,  # 1MB (<= 2MB limit)
        "contentType": "image/png",
    }
    response = await client.post(
        "/profile/me/media/presigned-urls", json=payload, headers=headers
    )
    assert response.status_code == 200
    data = response.json()
    assert "uploadUrl" in data
    assert "fileUrl" in data
    assert "companions/user_companion_123/albums/" in data["uploadUrl"]


async def test_presigned_url_invalid_image_size(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "assetType": "IMAGE",
        "sizeBytes": 3 * 1024 * 1024,  # 3MB (> 2MB limit)
        "contentType": "image/png",
    }
    response = await client.post(
        "/profile/me/media/presigned-urls", json=payload, headers=headers
    )
    assert response.status_code == 422
    assert "IMAGE size must not exceed 2MB" in response.json()["details"][0]["issue"]


async def test_presigned_url_valid_voice(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "assetType": "VOICE",
        "sizeBytes": 3 * 1024 * 1024,  # 3MB (<= 5MB limit)
        "durationSeconds": 25,  # <= 30s limit
        "contentType": "audio/mp3",
    }
    response = await client.post(
        "/profile/me/media/presigned-urls", json=payload, headers=headers
    )
    assert response.status_code == 200
    data = response.json()
    assert "companions/user_companion_123/voice_intro.mp3" in data["uploadUrl"]


async def test_presigned_url_invalid_voice_duration(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "assetType": "VOICE",
        "sizeBytes": 1 * 1024 * 1024,
        "durationSeconds": 35,  # > 30s limit
        "contentType": "audio/mp3",
    }
    response = await client.post(
        "/profile/me/media/presigned-urls", json=payload, headers=headers
    )
    assert response.status_code == 422
    assert (
        "durationSeconds must not exceed 30 seconds"
        in response.json()["details"][0]["issue"]
    )


async def test_search_companions_camel_case(client, integration_deps, db_session):
    scenario_cmd = integration_deps["scenario_cmd"]

    # 1. Add an active scenario with price 150

    # 2. Add an active scenario with price 150
    await scenario_cmd.create_scenario(
        companion_id="user_companion_123",
        title="Romantic Walk",
        description="A romantic walk in the city.",
        price=150,
        duration_minutes=60,
    )
    await db_session.commit()

    # 3. Test with correct camelCase query parameters
    # The starting price of user_companion_123 is 150.
    # Searching with minPrice=100 & maxPrice=200 should return the companion.
    response = await client.get(
        "/companions", params={"minPrice": 100, "maxPrice": 200, "pageSize": 5}
    )
    assert response.status_code == 200
    data = response.json()
    assert len(data["data"]) == 1
    assert data["data"][0]["companionId"] == "user_companion_123"
    assert data["pageSize"] == 5

    # 4. Test with price out of range
    # minPrice=200 should exclude the companion
    response = await client.get(
        "/companions", params={"minPrice": 200, "maxPrice": 300}
    )
    assert response.status_code == 200
    data = response.json()
    assert len(data["data"]) == 0

    # 5. Test with old snake_case parameters (which are now ignored by the API)
    # The API will see min_price but won't parse it because it expects minPrice.
    # So it won't apply the minPrice filter and user_companion_123 will still be returned.
    response = await client.get(
        "/companions", params={"minPrice": 100, "max_price": 100}
    )
    assert response.status_code == 200
    data = response.json()
    assert len(data["data"]) == 1
    assert data["data"][0]["companionId"] == "user_companion_123"


async def test_create_scenario(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "title": "Movie Date",
        "description": "Watch a movie together",
        "price": 200,
        "durationMinutes": 120,
    }
    response = await client.post("/profile/me/scenarios", json=payload, headers=headers)
    assert response.status_code == 201
    data = response.json()
    assert "scenarioId" in data


async def test_update_scenario(client, integration_deps, db_session):
    scenario_cmd = integration_deps["scenario_cmd"]
    scenario_id = await scenario_cmd.create_scenario(
        companion_id="user_companion_123",
        title="Walk in park",
        description="A nice walk",
        price=100,
        duration_minutes=60,
    )
    await db_session.commit()

    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "title": "Walk in park updated",
        "description": "A very nice walk",
        "price": 150,
        "durationMinutes": 120,
        "status": "INACTIVE",
    }
    response = await client.put(
        f"/profile/me/scenarios/{scenario_id}", json=payload, headers=headers
    )
    assert response.status_code == 200
    assert response.json() == {"success": True}


async def test_delete_scenario(client, integration_deps, db_session):
    scenario_cmd = integration_deps["scenario_cmd"]
    scenario_id = await scenario_cmd.create_scenario(
        companion_id="user_companion_123",
        title="To be deleted",
        description="Delete me",
        price=100,
        duration_minutes=60,
    )
    await db_session.commit()

    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    response = await client.delete(
        f"/profile/me/scenarios/{scenario_id}", headers=headers
    )
    assert response.status_code == 200
    assert response.json() == {"success": True}


async def test_create_and_update_my_profile_success(
    client, integration_deps, db_session
):
    # We will use a new user ID that doesn't have a profile yet
    new_user_id = "user_companion_456"
    headers = {"user-id": new_user_id, "user-role": "COMPANION"}

    # 1. Create Profile
    payload_create = {
        "displayName": "Mizuhara Chizuru",
        "bio": "I am a professional companion.",
        "availableCities": ["Tokyo", "Kyoto"],
    }
    response = await client.post(
        "/profile/me",
        json=payload_create,
        headers=headers,
    )
    assert response.status_code == 201
    assert response.json()["companionId"] == new_user_id

    # Verify in DB
    profile_repo = integration_deps["profile_repo"]
    profile = await profile_repo.find_by_id(new_user_id)
    assert profile is not None
    assert profile.display_name == "Mizuhara Chizuru"
    assert profile.status == "APPROVED"

    # 2. Update Profile
    payload_update = {
        "displayName": "Mizuhara Chizuru Updated",
        "bio": "Updated bio.",
        "availableCities": ["Tokyo", "Kyoto", "Osaka"],
        "avatarUrl": "https://s3.rentgf.com/avatars/chizuru.jpg",
    }
    response = await client.put(
        "/profile/me",
        json=payload_update,
        headers=headers,
    )
    assert response.status_code == 200
    assert response.json() == {"success": True}

    # Verify update in DB
    updated_profile = await profile_repo.find_by_id(new_user_id)
    assert updated_profile.display_name == "Mizuhara Chizuru Updated"
    assert updated_profile.bio == "Updated bio."
    assert (
        str(updated_profile.avatar_url) == "https://s3.rentgf.com/avatars/chizuru.jpg"
    )


async def test_client_profile_creation_without_bio(client, integration_deps):
    new_user_id = "user_client_789"
    headers = {"user-id": new_user_id, "user-role": "CLIENT"}

    # 1. Create client profile (bio omitted)
    payload = {
        "displayName": "Standard Client",
        "availableCities": ["Hanoi"],
    }
    response = await client.post("/profile/me", json=payload, headers=headers)
    assert response.status_code == 201

    # 2. Query profile
    response = await client.get("/profile/me", headers=headers)
    assert response.status_code == 200
    data = response.json()
    assert data["displayName"] == "Standard Client"
    assert data["role"] == "CLIENT"
    assert data["bio"] == ""


async def test_client_blocked_from_companion_apis(client):
    headers = {"user-id": "user_client_789", "user-role": "CLIENT"}

    # Try creating scenario
    payload_scenario = {
        "title": "Park Date",
        "description": "Walk in the park",
        "price": 100,
        "durationMinutes": 60,
    }
    response = await client.post(
        "/profile/me/scenarios", json=payload_scenario, headers=headers
    )
    assert response.status_code == 403

    # Try requesting presigned url for VOICE
    payload_url = {
        "assetType": "VOICE",
        "sizeBytes": 1000,
        "durationSeconds": 10,
        "contentType": "audio/mp3",
    }
    response = await client.post(
        "/profile/me/media/presigned-urls", json=payload_url, headers=headers
    )
    assert response.status_code == 403


async def test_search_companions_excludes_clients(client, integration_deps, db_session):
    # Ensure user_companion_123 is APPROVED to avoid contamination from previous tests
    profile_repo = integration_deps["profile_repo"]
    profile = await profile_repo.find_by_id("user_companion_123")
    if profile:
        profile.status = "APPROVED"
        await profile_repo.save(profile)
        await db_session.commit()

    # Seed a client profile
    profile_cmd = integration_deps["profile_cmd"]
    await profile_cmd.create_profile(
        companion_id="user_client_only",
        user_id="user_client_only",
        display_name="Client User Only",
        available_cities=["Hanoi"],
        role="CLIENT",
    )
    await db_session.commit()

    # Search companions
    response = await client.get("/companions?city=Hanoi")
    assert response.status_code == 200
    data = response.json()["data"]

    # Verify client user is not in the list (only user_companion_123)
    client_ids_found = [c["companionId"] for c in data]
    assert "user_client_only" not in client_ids_found
    assert "user_companion_123" in client_ids_found


async def test_identity_listener_upgrades_role(
    integration_deps, db_session, monkeypatch
):
    # Seed a client profile
    profile_repo = integration_deps["profile_repo"]
    profile_cmd = integration_deps["profile_cmd"]
    await profile_cmd.create_profile(
        companion_id="user_to_upgrade_111",
        user_id="user_to_upgrade_111",
        display_name="Client To Upgrade",
        available_cities=["Hanoi"],
        role="CLIENT",
    )
    await db_session.commit()

    # Mock Message and AIOKafkaConsumer
    class MockMessage:
        def __init__(self, value):
            self.value = value

    class MockConsumer:
        def __init__(self, messages):
            self.messages = messages

        async def start(self):
            pass

        async def stop(self):
            pass

        def __aiter__(self):
            return self

        async def __anext__(self):
            if not self.messages:
                raise StopAsyncIteration
            return self.messages.pop(0)

    # Mock settings and import targeting main listener module or identity_listener.py
    import internal.interfaces.kafka.identity_listener as listener_module

    from contextlib import asynccontextmanager

    @asynccontextmanager
    async def mock_session_local():
        yield db_session

    monkeypatch.setattr(listener_module, "SessionLocal", mock_session_local)

    # Mock AIOKafkaConsumer in the listener module
    mock_event = {
        "specversion": "1.0",
        "type": "identity.role-upgraded.v1",
        "data": {
            "userId": "user_to_upgrade_111",
            "oldRole": "CLIENT",
            "newRole": "COMPANION",
        },
    }
    monkeypatch.setattr(
        listener_module,
        "AIOKafkaConsumer",
        lambda *args, **kwargs: MockConsumer([MockMessage(mock_event)]),
    )

    # Execute listener
    from internal.interfaces.kafka.identity_listener import IdentityEventListener

    listener = IdentityEventListener()

    # Execute the internal running loop directly (it will run, process the mock event, and exit)
    await listener._run()

    # Verify user upgraded in DB
    db_session.expire_all()
    profile = await profile_repo.find_by_id("user_to_upgrade_111")
    assert profile is not None
    assert profile.role == "COMPANION"
    assert profile.status == "APPROVED"
