import os
import logging
from fastapi import FastAPI, Depends
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker
from pydantic_settings import BaseSettings, SettingsConfigDict

# Domain & Application Layer Imports
from internal.infrastructure.persistence import (
    Base,
    DisputeRepository,
    SagaStateRepository,
)
from internal.infrastructure.persistence.db_cleanup_worker import DbCleanupWorker
from internal.infrastructure.broker import (
    DatabaseEventPublisher,
    OutboxPublisherWorker,
    SagaRetryWorker,
    DisputeEventConsumer,
)
from internal.infrastructure.adapters import (
    MockFinanceAdapter,
    MockInteractionAdapter,
)
from internal.application.saga import (
    DisputeRefundSagaOrchestrator,
    DisputePayoutSagaOrchestrator,
)
from internal.application.command import DisputeCommandService
from internal.application.query import DisputeQueryService

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("bootstrap")


class Settings(BaseSettings):
    SERVER_PORT: int = 8082
    GRPC_PORT: int = 50051
    APP_ENV: str = "development"

    DB_HOST: str = "localhost"
    DB_PORT: int = 5432
    DB_USER: str = "dispute_admin"
    DB_PASSWORD: str = "super_secure_password"
    DB_NAME: str = "dispute_service"
    DB_SSLMODE: str = "disable"

    KAFKA_BROKERS: str = "localhost:9092"
    KAFKA_TOPIC_DISPUTE: str = "dispute.events"

    OUTBOX_POLLING_INTERVAL_MS: int = 500
    OUTBOX_BATCH_SIZE: int = 50

    SAGA_RETRY_INTERVAL_SECONDS: float = 5.0

    USE_MOCKS: bool = True
    FINANCE_SERVICE_ADDR: str = "localhost:50052"
    INTERACTION_SERVICE_ADDR: str = "localhost:50053"

    DB_CLEANUP_INTERVAL_MINUTES: int = 30
    PROCESSED_EVENTS_RETENTION_DAYS: int = 7
    OUTBOX_RETENTION_DAYS: int = 1

    model_config = SettingsConfigDict(env_file=".env", extra="ignore")


settings = Settings()

# --- Database Setup ---
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


# --- Dependency Assembly (DI) ---
def bootstrap_services(db_session: AsyncSession):
    # Repositories
    dispute_repo = DisputeRepository(db_session)
    saga_repo = SagaStateRepository(db_session)

    # Event Publisher (Atomic Outbox table insertions)
    event_publisher = DatabaseEventPublisher(db_session)

    # Inter-service adapters
    if settings.USE_MOCKS:
        finance_port = MockFinanceAdapter()
        interaction_port = MockInteractionAdapter()
    else:
        from internal.infrastructure.adapters import (
            gRPCFinanceAdapter,
            gRPCInteractionAdapter,
        )

        finance_port = gRPCFinanceAdapter(settings.FINANCE_SERVICE_ADDR)
        interaction_port = gRPCInteractionAdapter(settings.INTERACTION_SERVICE_ADDR)

    # SAGA Orchestrators
    refund_saga_orchestrator = DisputeRefundSagaOrchestrator(
        saga_repo=saga_repo,
        finance_port=finance_port,
        interaction_port=interaction_port,
        event_publisher=event_publisher,
    )
    payout_saga_orchestrator = DisputePayoutSagaOrchestrator(
        saga_repo=saga_repo,
        finance_port=finance_port,
        interaction_port=interaction_port,
        event_publisher=event_publisher,
    )

    # Command Service
    cmd_service = DisputeCommandService(
        dispute_repo=dispute_repo,
        saga_repo=saga_repo,
        event_publisher=event_publisher,
        refund_saga_orchestrator=refund_saga_orchestrator,
        payout_saga_orchestrator=payout_saga_orchestrator,
    )

    # Query Service
    query_service = DisputeQueryService(
        dispute_repo=dispute_repo,
        saga_repo=saga_repo,
    )

    return cmd_service, query_service


# --- Dependency Injection Functions ---
async def get_db_session():
    async with SessionLocal() as db:
        yield db


async def get_services(db: AsyncSession = Depends(get_db_session)):
    return bootstrap_services(db)


async def get_query_service(services=Depends(get_services)) -> DisputeQueryService:
    return services[1]


# --- Workers & Consumers Setup ---
db_cleanup_worker = DbCleanupWorker(
    session_factory=SessionLocal,
    cleanup_interval_minutes=settings.DB_CLEANUP_INTERVAL_MINUTES,
    processed_events_retention_days=settings.PROCESSED_EVENTS_RETENTION_DAYS,
    outbox_retention_days=settings.OUTBOX_RETENTION_DAYS,
)

saga_retry_worker = SagaRetryWorker(
    session_factory=SessionLocal,
    polling_interval_seconds=settings.SAGA_RETRY_INTERVAL_SECONDS,
)

outbox_worker = OutboxPublisherWorker(
    session_factory=SessionLocal,
    kafka_brokers=settings.KAFKA_BROKERS,
    topic=settings.KAFKA_TOPIC_DISPUTE,
    polling_interval_ms=settings.OUTBOX_POLLING_INTERVAL_MS,
    batch_size=settings.OUTBOX_BATCH_SIZE,
)

event_consumer = DisputeEventConsumer(
    session_factory=SessionLocal,
    kafka_brokers=settings.KAFKA_BROKERS,
    topics=["finance-replies", "interaction-replies"],
)

# --- FastAPI App Setup ---
app = FastAPI(
    title="Dispute Resolution Service REST API",
    description="REST endpoints for Admin Dashboard to manage disputes and view SAGA states.",
    version="1.0.0",
    openapi_url="/api/openapi.json",
)

from internal.interfaces.http.router import router as http_router  # noqa: E402

app.include_router(http_router)

import internal.interfaces.http.errors as http_errors  # noqa: E402
from internal.domain.errors import DomainError  # noqa: E402
from fastapi.exceptions import RequestValidationError  # noqa: E402
from starlette.exceptions import HTTPException as StarletteHTTPException  # noqa: E402

app.add_exception_handler(DomainError, http_errors.domain_error_handler)
app.add_exception_handler(StarletteHTTPException, http_errors.http_exception_handler)
app.add_exception_handler(RequestValidationError, http_errors.validation_exception_handler)
