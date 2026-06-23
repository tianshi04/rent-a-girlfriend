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
            intro_text="Perfect rental girlfriend",
            available_cities=["Hanoi"],
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
    profile_cmd = integration_deps["profile_cmd"]
    scenario_cmd = integration_deps["scenario_cmd"]

    # 1. Approve the companion profile
    await profile_cmd.approve_profile("user_companion_123", "admin_user_99")

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
