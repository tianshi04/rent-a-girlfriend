import os
import logging
import asyncio
from fastapi import FastAPI, Depends
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker
from pydantic_settings import BaseSettings, SettingsConfigDict
from internal.infrastructure.persistence import (
    Base,
    CompanionProfileRepository,
    ScenarioRepository,
    MediaAssetRepository,
)
from internal.infrastructure.storage import S3Storage
from internal.infrastructure.broker import DatabaseEventPublisher, OutboxPublisherWorker
from internal.infrastructure.persistence.db_cleanup_worker import DbCleanupWorker
from internal.application.command import (
    ProfileCommandService,
    ScenarioCommandService,
    MediaCommandService,
)
from internal.application.query import ProfileQueryService

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("bootstrap")


class Settings(BaseSettings):
    SERVER_PORT: int = 8080
    GRPC_PORT: int = 50051
    APP_ENV: str = "development"

    DB_HOST: str = "localhost"
    DB_PORT: int = 5432
    DB_USER: str = "profile_admin"
    DB_PASSWORD: str = "super_secure_password"
    DB_NAME: str = "profile_service"
    DB_SSLMODE: str = "disable"

    S3_ENDPOINT_URL: str = "http://localhost:9000"
    S3_REGION: str = "us-east-1"
    S3_ACCESS_KEY_ID: str = "minio_admin"
    S3_SECRET_ACCESS_KEY: str = "minio_secure_password"
    S3_BUCKET_NAME: str = "rentgf-media"

    KAFKA_BROKERS: str = "localhost:9092"
    KAFKA_TOPIC_PROFILE: str = "profile.events"

    OUTBOX_POLLING_INTERVAL_MS: int = 500
    OUTBOX_BATCH_SIZE: int = 50

    DB_CLEANUP_INTERVAL_MINUTES: int = 30
    OUTBOX_RETENTION_DAYS: int = 1

    model_config = SettingsConfigDict(env_file=".env", extra="ignore")


settings = Settings()

# --- Database Setup ---
# Use in-memory sqlite if running tests, otherwise standard postgresql
if os.environ.get("TESTING") == "1":
    DATABASE_URL = "sqlite+aiosqlite:///:memory:"
else:
    DATABASE_URL = f"postgresql+asyncpg://{settings.DB_USER}:{settings.DB_PASSWORD}@{settings.DB_HOST}:{settings.DB_PORT}/{settings.DB_NAME}"

logger.info(
    f"Connecting to database at {DATABASE_URL.split('@')[-1] if '@' in DATABASE_URL else DATABASE_URL}"
)
engine = create_async_engine(DATABASE_URL, echo=False)
SessionLocal = async_sessionmaker(
    autocommit=False, autoflush=False, bind=engine, class_=AsyncSession
)


# Create tables asynchronously in development/production
async def init_db():
    try:
        async with engine.begin() as conn:
            await conn.run_sync(Base.metadata.create_all)
        logger.info("Database tables initialized successfully.")
    except Exception as e:
        logger.error(
            f"Failed to auto-create database tables: {e}. Is database running?"
        )


if os.environ.get("TESTING") != "1":
    try:
        asyncio.run(init_db())
    except Exception as e:
        logger.warning(f"Could not run database initialization at import time: {e}")

# --- Storage Adapter Setup ---
storage_adapter = S3Storage(
    bucket_name=settings.S3_BUCKET_NAME,
    region_name=settings.S3_REGION,
    access_key_id=settings.S3_ACCESS_KEY_ID,
    secret_access_key=settings.S3_SECRET_ACCESS_KEY,
    endpoint_url=settings.S3_ENDPOINT_URL if settings.S3_ENDPOINT_URL else None,
)


# --- Dependency Assembly (DI) ---
# Create scoped services helpers
def bootstrap_services(db_session: AsyncSession):
    # Repositories
    profile_repo = CompanionProfileRepository(db_session)
    scenario_repo = ScenarioRepository(db_session)
    media_repo = MediaAssetRepository(db_session)

    # Event Publisher (Atomic Outbox table insertions)
    event_publisher = DatabaseEventPublisher(db_session)

    # Application Commands
    profile_cmd = ProfileCommandService(profile_repo, event_publisher)
    scenario_cmd = ScenarioCommandService(profile_repo, scenario_repo, event_publisher)
    media_cmd = MediaCommandService(
        profile_repo, media_repo, storage_adapter, event_publisher
    )

    # Application Queries
    query_service = ProfileQueryService(profile_repo, scenario_repo, media_repo)

    return profile_cmd, scenario_cmd, media_cmd, query_service


# --- Dependency Injection Functions ---
async def get_db_session():
    async with SessionLocal() as db:
        yield db


async def get_services(db: AsyncSession = Depends(get_db_session)):
    return bootstrap_services(db)


async def get_query_service(services=Depends(get_services)) -> ProfileQueryService:
    return services[3]


async def get_media_cmd(services=Depends(get_services)) -> MediaCommandService:
    return services[2]


# --- Outbox & Cleanup Workers Setup ---
db_cleanup_worker = DbCleanupWorker(
    session_factory=SessionLocal,
    cleanup_interval_minutes=settings.DB_CLEANUP_INTERVAL_MINUTES,
    outbox_retention_days=settings.OUTBOX_RETENTION_DAYS,
)

outbox_worker = OutboxPublisherWorker(
    session_factory=SessionLocal,
    kafka_brokers=settings.KAFKA_BROKERS,
    topic=settings.KAFKA_TOPIC_PROFILE,
    polling_interval_ms=settings.OUTBOX_POLLING_INTERVAL_MS,
    batch_size=settings.OUTBOX_BATCH_SIZE,
)

# --- FastAPI App Setup ---
app = FastAPI(
    title="Profile & Catalogue Service REST Query API",
    description="Query catalogue and manage profiles.",
    version="1.0.0",
    openapi_url="/api/openapi.json",
)

from internal.interfaces.http.router import router  # noqa: E402

app.include_router(router)


@app.get("/health", tags=["System"])
async def health_check():
    return {"status": "ok", "service": "profile-service"}
