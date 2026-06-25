import os
import pytest
import pytest_asyncio
from testcontainers.postgres import PostgresContainer
from testcontainers.kafka import KafkaContainer
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

# Setup testing env var
os.environ["TESTING"] = "1"

from internal.bootstrap import Base, app
from internal.infrastructure.persistence import (
    UserProfileRepository,
    CompanionProfileRepository,
    ScenarioRepository,
    MediaAssetRepository,
)
from internal.infrastructure.storage import S3Storage
from internal.infrastructure.broker import DatabaseEventPublisher
from internal.application.command import (
    ProfileCommandService,
    ScenarioCommandService,
    MediaCommandService,
)
from internal.application.query import ProfileQueryService
from internal.bootstrap import (
    get_query_service,
    get_media_cmd,
    get_scenario_cmd,
    get_profile_cmd,
)


@pytest.fixture(scope="session")
def postgres():
    with PostgresContainer("postgres:15-alpine") as postgres:
        yield postgres


@pytest.fixture(scope="session")
def kafka():
    with KafkaContainer("confluentinc/cp-kafka:7.4.0") as kafka:
        yield kafka


@pytest_asyncio.fixture
async def db_engine(postgres):
    connection_url = postgres.get_connection_url()
    if connection_url.startswith("postgresql://"):
        connection_url = connection_url.replace(
            "postgresql://", "postgresql+asyncpg://", 1
        )
    elif "postgresql+psycopg2://" in connection_url:
        connection_url = connection_url.replace(
            "postgresql+psycopg2://", "postgresql+asyncpg://", 1
        )

    engine = create_async_engine(connection_url)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    await engine.dispose()


@pytest.fixture
def TestSessionLocal(db_engine):
    return async_sessionmaker(
        autocommit=False, autoflush=False, bind=db_engine, class_=AsyncSession
    )


@pytest_asyncio.fixture
async def db_session(TestSessionLocal):
    async with TestSessionLocal() as session:
        yield session


@pytest.fixture(autouse=True)
def integration_deps(db_session, kafka):
    user_profile_repo = UserProfileRepository(db_session)
    profile_repo = CompanionProfileRepository(db_session)
    scenario_repo = ScenarioRepository(db_session)
    media_repo = MediaAssetRepository(db_session)
    event_publisher = DatabaseEventPublisher(db_session)

    storage_mock = S3Storage(
        bucket_name="test-bucket",
        region_name="us-east-1",
        access_key_id="test",
        secret_access_key="test",
        endpoint_url="http://localhost:9000",
    )
    storage_mock.generate_presigned_put_url = lambda key, content_type, content_length: (
        f"https://mock-s3.com/{key}"
    )
    storage_mock.generate_presigned_get_url = lambda key, expires_in=300: (
        f"https://mock-s3.com/{key}?expires={expires_in}"
    )

    profile_cmd = ProfileCommandService(
        user_profile_repo, profile_repo, event_publisher
    )
    scenario_cmd = ScenarioCommandService(profile_repo, scenario_repo, event_publisher)
    media_cmd = MediaCommandService(
        profile_repo, media_repo, storage_mock, event_publisher
    )
    query_service = ProfileQueryService(
        user_profile_repo, profile_repo, scenario_repo, media_repo, storage_mock
    )

    app.dependency_overrides[get_query_service] = lambda: query_service
    app.dependency_overrides[get_media_cmd] = lambda: media_cmd
    app.dependency_overrides[get_scenario_cmd] = lambda: scenario_cmd
    app.dependency_overrides[get_profile_cmd] = lambda: profile_cmd

    return {
        "profile_cmd": profile_cmd,
        "scenario_cmd": scenario_cmd,
        "media_cmd": media_cmd,
        "query_service": query_service,
        "profile_repo": profile_repo,
        "user_profile_repo": user_profile_repo,
    }
