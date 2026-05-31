import os
import sys
import pytest
import pytest_asyncio

# Force testing environment variable
os.environ["TESTING"] = "1"

# Resolve paths
root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
if root_dir not in sys.path:
    sys.path.insert(0, root_dir)
grpc_dir = os.path.join(root_dir, "gen")
if grpc_dir not in sys.path:
    sys.path.insert(0, grpc_dir)

from sqlalchemy.ext.asyncio import create_async_engine, async_sessionmaker, AsyncSession
from sqlalchemy.pool import StaticPool

# Now we import bootstrap
import internal.bootstrap
from internal.bootstrap import Base, app

# Patch engine to use StaticPool for SQLite in-memory sharing
patched_engine = create_async_engine(
    "sqlite+aiosqlite:///:memory:",
    poolclass=StaticPool,
    connect_args={"check_same_thread": False},
    echo=False,
)
patched_sessionmaker = async_sessionmaker(
    autocommit=False, autoflush=False, bind=patched_engine, class_=AsyncSession
)

# Overwrite in bootstrap module
internal.bootstrap.engine = patched_engine
internal.bootstrap.SessionLocal = patched_sessionmaker


# Also override the get_db_session dependency function
async def override_get_db_session():
    async with patched_sessionmaker() as db:
        yield db


internal.bootstrap.get_db_session = override_get_db_session
app.dependency_overrides[internal.bootstrap.get_db_session] = override_get_db_session


@pytest_asyncio.fixture(autouse=True)
async def init_test_db():
    """Initializes schema once per test function."""
    async with patched_engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield
    async with patched_engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)


@pytest_asyncio.fixture
async def db_session():
    """Provides a transactional database session for setup and verification."""
    async with patched_sessionmaker() as session:
        yield session
        # Ensure we rollback any uncommitted changes to keep tests clean
        await session.rollback()


@pytest.fixture
def test_session_factory():
    """Provides the patched sessionmaker to be passed to services/servicers."""
    return patched_sessionmaker


@pytest.fixture
def integration_deps(db_session):
    """Initializes command and query services with the active test session."""
    from internal.bootstrap import bootstrap_services

    cmd_service, query_service = bootstrap_services(db_session)
    return {
        "cmd_service": cmd_service,
        "query_service": query_service,
        "session": db_session,
    }
