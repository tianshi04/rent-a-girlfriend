import pytest
from httpx import AsyncClient, ASGITransport
from internal.bootstrap import app

pytestmark = pytest.mark.asyncio

@pytest.fixture
async def client():
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://testserver/api/v1") as ac:
        yield ac

@pytest.fixture
async def root_client():
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://testserver") as ac:
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
            available_cities=["Hanoi"]
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
        "sizeBytes": 1 * 1024 * 1024, # 1MB (<= 2MB limit)
        "contentType": "image/png"
    }
    response = await client.post("/profile/me/media/presigned-urls", json=payload, headers=headers)
    assert response.status_code == 200
    data = response.json()
    assert "uploadUrl" in data
    assert "fileUrl" in data
    assert "companions/user_companion_123/albums/" in data["uploadUrl"]

async def test_presigned_url_invalid_image_size(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "assetType": "IMAGE",
        "sizeBytes": 3 * 1024 * 1024, # 3MB (> 2MB limit)
        "contentType": "image/png"
    }
    response = await client.post("/profile/me/media/presigned-urls", json=payload, headers=headers)
    assert response.status_code == 422
    assert "IMAGE size must not exceed 2MB" in response.json()["detail"][0]["msg"]

async def test_presigned_url_valid_voice(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "assetType": "VOICE",
        "sizeBytes": 3 * 1024 * 1024, # 3MB (<= 5MB limit)
        "durationSeconds": 25, # <= 30s limit
        "contentType": "audio/mp3"
    }
    response = await client.post("/profile/me/media/presigned-urls", json=payload, headers=headers)
    assert response.status_code == 200
    data = response.json()
    assert "companions/user_companion_123/voice_intro.mp3" in data["uploadUrl"]

async def test_presigned_url_invalid_voice_duration(client):
    headers = {"user-id": "user_companion_123", "user-role": "COMPANION"}
    payload = {
        "assetType": "VOICE",
        "sizeBytes": 1 * 1024 * 1024,
        "durationSeconds": 35, # > 30s limit
        "contentType": "audio/mp3"
    }
    response = await client.post("/profile/me/media/presigned-urls", json=payload, headers=headers)
    assert response.status_code == 422
    assert "durationSeconds must not exceed 30 seconds" in response.json()["detail"][0]["msg"]
