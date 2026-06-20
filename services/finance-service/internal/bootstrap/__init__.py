import os
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI, Depends
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker
from pydantic_settings import BaseSettings, SettingsConfigDict
from internal.infrastructure.persistence import (
    Base,
    WalletRepository,
    EscrowRepository,
    TransactionRepository,
)
from internal.infrastructure.payment.vnpay import VNPayAdapter
from internal.infrastructure.broker import DatabaseEventPublisher, OutboxPublisherWorker
from internal.infrastructure.persistence.db_cleanup_worker import DbCleanupWorker
from internal.application.command.finance import FinanceCommandService

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("bootstrap")


class Settings(BaseSettings):
    SERVER_PORT: int = 8080
    GRPC_PORT: int = 50051
    APP_ENV: str = "development"

    DB_HOST: str = "localhost"
    DB_PORT: int = 5432
    DB_USER: str = "finance_admin"
    DB_PASSWORD: str = "super_secure_password"
    DB_NAME: str = "finance_service"
    DB_SSLMODE: str = "disable"

    VNPAY_TMN_CODE: str = "2QF59G25"  # VNPay Sandbox default
    VNPAY_HASH_SECRET: str = "A5D7F1G3J5K7M9P1R3T5V7X9Z1B3D5F7"  # VNPay Sandbox default
    VNPAY_URL: str = "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html"
    VNPAY_RETURN_URL: str = "http://localhost:8080/api/v1/finance/vnpay-return"

    KAFKA_BROKERS: str = "localhost:9092"
    KAFKA_TOPIC_FINANCE: str = "finance.events"
    KAFKA_TOPIC_IDENTITY: str = "identity.events"  # Lắng nghe UserRegistered từ đây
    KAFKA_TOPIC_BOOKING: str = "booking.events"  # Lắng nghe BookingRequested từ đây
    KAFKA_GROUP_ID: str = "finance-service-group"

    OUTBOX_POLLING_INTERVAL_MS: int = 500
    OUTBOX_BATCH_SIZE: int = 50

    DB_CLEANUP_INTERVAL_MINUTES: int = 30
    OUTBOX_RETENTION_DAYS: int = 1

    model_config = SettingsConfigDict(env_file=".env", extra="ignore")


settings = Settings()

# --- Database Setup ---
# Sử dụng SQLite in-memory cho testing hoặc Postgres cho production
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


# Create tables asynchronously
async def init_db():
    try:
        async with engine.begin() as conn:
            await conn.run_sync(Base.metadata.create_all)
        logger.info("Database tables initialized successfully.")
    except Exception as e:
        logger.error(
            f"Failed to auto-create database tables: {e}. Is database running?"
        )


# --- VNPay Adapter Setup ---
vnpay_adapter = VNPayAdapter(
    tmn_code=settings.VNPAY_TMN_CODE,
    hash_secret=settings.VNPAY_HASH_SECRET,
    payment_url=settings.VNPAY_URL,
    return_url=settings.VNPAY_RETURN_URL,
)


# --- Dependency Assembly (DI) ---
def bootstrap_services(db_session: AsyncSession) -> FinanceCommandService:
    # Repositories
    wallet_repo = WalletRepository(db_session)
    escrow_repo = EscrowRepository(db_session)
    transaction_repo = TransactionRepository(db_session)

    # Event Publisher (Atomic Outbox insertions)
    event_publisher = DatabaseEventPublisher(db_session)

    # Application Command Service
    finance_cmd = FinanceCommandService(
        wallet_repo=wallet_repo,
        escrow_repo=escrow_repo,
        transaction_repo=transaction_repo,
        event_publisher=event_publisher,
        vnpay_adapter=vnpay_adapter,
        session=db_session,
    )

    return finance_cmd


# --- Dependency Injection Functions ---
async def get_db_session():
    async with SessionLocal() as db:
        try:
            yield db
            await db.commit()
        except Exception:
            await db.rollback()
            raise


async def get_finance_cmd(
    db: AsyncSession = Depends(get_db_session),
) -> FinanceCommandService:
    return bootstrap_services(db)


# --- Outbox & Cleanup Workers Setup ---
db_cleanup_worker = DbCleanupWorker(
    session_factory=SessionLocal,
    cleanup_interval_minutes=settings.DB_CLEANUP_INTERVAL_MINUTES,
    outbox_retention_days=settings.OUTBOX_RETENTION_DAYS,
)

outbox_worker = OutboxPublisherWorker(
    session_factory=SessionLocal,
    kafka_brokers=settings.KAFKA_BROKERS,
    topic=settings.KAFKA_TOPIC_FINANCE,
    polling_interval_ms=settings.OUTBOX_POLLING_INTERVAL_MS,
    batch_size=settings.OUTBOX_BATCH_SIZE,
)

# --- Lifespan Setup ---


@asynccontextmanager
async def lifespan(app: FastAPI):
    if os.environ.get("TESTING") != "1":
        await init_db()
    yield


# --- FastAPI App Setup ---
app = FastAPI(
    title="Finance Service REST API",
    description="REST API for Kano-Coin Top-up & Webhooks",
    version="1.0.0",
    lifespan=lifespan,
)

# router will be imported later in controllers.py to avoid circular dependencies
from internal.interfaces.http.router import router  # noqa: E402

app.include_router(router)


@app.get("/health", tags=["System"])
async def health_check():
    return {"status": "ok", "service": "finance-service"}


import internal.interfaces.http.errors as http_errors  # noqa: E402
from internal.domain.errors import DomainError  # noqa: E402
from fastapi.exceptions import RequestValidationError  # noqa: E402
from starlette.exceptions import HTTPException as StarletteHTTPException  # noqa: E402

app.add_exception_handler(DomainError, http_errors.domain_error_handler)
app.add_exception_handler(StarletteHTTPException, http_errors.http_exception_handler)
app.add_exception_handler(
    RequestValidationError, http_errors.validation_exception_handler
)
