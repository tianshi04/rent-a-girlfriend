"""
conftest.py — Root test configuration for Finance Service.
"""

import os
import pytest
import pytest_asyncio
from testcontainers.postgres import PostgresContainer
from testcontainers.kafka import KafkaContainer
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

os.environ["TESTING"] = "1"

from internal.infrastructure.persistence.models import Base


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
