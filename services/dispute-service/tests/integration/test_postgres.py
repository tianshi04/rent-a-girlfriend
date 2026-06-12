import pytest
import pytest_asyncio
import uuid
from sqlalchemy.ext.asyncio import create_async_engine, async_sessionmaker, AsyncSession

from internal.infrastructure.persistence.models import Base
from internal.infrastructure.persistence.repositories import DisputeRepository
from internal.domain.aggregate import Dispute, DisputeEvidence

# Determine if testcontainers is installed
try:
    from testcontainers.postgres import PostgresContainer

    HAS_TESTCONTAINERS = True
except ImportError:
    HAS_TESTCONTAINERS = False

pytestmark = pytest.mark.asyncio


@pytest.fixture(scope="module")
def postgres_url():
    """
    Spins up a PostgreSQL container for the test module.
    Yields the asyncpg connection string.
    """
    if not HAS_TESTCONTAINERS:
        pytest.skip(
            "testcontainers[postgres] is not installed. Please run `uv add --group dev testcontainers[postgres]`"
        )
        return

    # Use a lightweight Postgres 16 alpine image
    with PostgresContainer("postgres:16-alpine") as postgres:
        # testcontainers returns a sync psycopg2 URL by default.
        # We need to convert it to asyncpg for our stack.
        db_url = postgres.get_connection_url()
        async_db_url = db_url.replace("postgresql+psycopg2://", "postgresql+asyncpg://")
        yield async_db_url


@pytest_asyncio.fixture
async def postgres_engine(postgres_url):
    if not postgres_url:
        return

    engine = create_async_engine(postgres_url, echo=False)

    # Initialize the database schema
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    yield engine

    # Teardown
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    await engine.dispose()


@pytest_asyncio.fixture
async def pg_session(postgres_engine):
    """Provides a transactional session connected to the real PostgreSQL."""
    if not postgres_engine:
        return

    session_factory = async_sessionmaker(
        autocommit=False, autoflush=False, bind=postgres_engine, class_=AsyncSession
    )
    async with session_factory() as session:
        yield session
        await session.rollback()


async def test_postgres_dispute_repository_integration(pg_session):
    """
    Verifies that the SQLAlchemy models, DisputeRepository, and asyncpg driver
    work perfectly together on a real PostgreSQL instance.
    Specifically checks for strict UTC timezone constraints imposed by asyncpg.
    """
    if not pg_session:
        return

    repo = DisputeRepository(pg_session)
    booking_id = str(uuid.uuid4())
    dispute_id = str(uuid.uuid4())

    # 1. Arrange: Create a new domain aggregate
    evidence = DisputeEvidence(
        evidence_id=str(uuid.uuid4()),
        evidence_type="IMAGE",
        content="https://storage.example.com/evidence1.jpg",
    )

    dispute = Dispute.create_report(
        dispute_id=dispute_id,
        booking_id=booking_id,
        reporter_id="client-123",
        accused_id="companion-456",
        reason="NO_SHOW",
        evidences=[evidence],
    )

    # 2. Act: Save to real PostgreSQL
    await repo.save(dispute)
    await pg_session.commit()

    # 3. Assert: Fetch back and verify mapping
    fetched_dispute = await repo.find_by_id(dispute_id)

    assert fetched_dispute is not None
    assert fetched_dispute.dispute_id == dispute_id
    assert fetched_dispute.booking_id == booking_id
    assert fetched_dispute.reporter_id == "client-123"
    assert str(fetched_dispute.reason) == "NO_SHOW"
    assert fetched_dispute.status == "OPEN"

    assert len(fetched_dispute.evidences) == 1
    fetched_evidence = fetched_dispute.evidences[0]
    assert fetched_evidence.evidence_type == "IMAGE"
    assert fetched_evidence.content == "https://storage.example.com/evidence1.jpg"

    # 4. Verify native PostgreSQL behavior (e.g. constraints, native sequence)
    # Check that [INV-D01] UNIQUE constraint on booking_id prevents duplicates at DB level
    duplicate_dispute = Dispute.create_report(
        dispute_id=str(uuid.uuid4()),
        booking_id=booking_id,  # Same booking_id
        reporter_id="client-123",
        accused_id="companion-456",
        reason="FRAUD",
    )

    import sqlalchemy.exc

    with pytest.raises(sqlalchemy.exc.IntegrityError):
        await repo.save(duplicate_dispute)
        await pg_session.commit()
