from datetime import datetime, timezone
from sqlalchemy import (
    Column,
    String,
    Integer,
    Float,
    Text,
    Boolean,
    DateTime,
    ForeignKey,
)
from sqlalchemy.orm import declarative_base, relationship

Base = declarative_base()


def utcnow_naive():
    """
    Returns a timezone-naive UTC datetime.
    Required because standard Column(DateTime) columns are TIMESTAMP WITHOUT TIME ZONE,
    and asyncpg strictly prohibits mixing offset-naive and offset-aware datetimes.
    """
    return datetime.now(timezone.utc).replace(tzinfo=None)


class DisputeModel(Base):
    __tablename__ = "disputes"

    dispute_id = Column(String(36), primary_key=True)
    booking_id = Column(String(36), unique=True, nullable=False, index=True)
    reporter_id = Column(String(36), nullable=False, index=True)
    accused_id = Column(String(36), nullable=False, index=True)
    reason = Column(String(50), nullable=False)
    status = Column(String(30), nullable=False, index=True)
    admin_id = Column(String(36), nullable=True, index=True)
    resolution = Column(String(50), nullable=True)
    notes = Column(Text, nullable=True)
    version = Column(Integer, default=1, nullable=False)
    created_at = Column(DateTime, default=utcnow_naive)
    updated_at = Column(DateTime, default=utcnow_naive, onupdate=utcnow_naive)

    __mapper_args__ = {"version_id_col": version}

    evidences = relationship(
        "DisputeEvidenceModel", back_populates="dispute", cascade="all, delete-orphan"
    )


class DisputeEvidenceModel(Base):
    __tablename__ = "dispute_evidences"

    evidence_id = Column(String(36), primary_key=True)
    dispute_id = Column(
        String(36),
        ForeignKey("disputes.dispute_id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    evidence_type = Column(String(30), nullable=False)  # TEXT, IMAGE
    content = Column(Text, nullable=False)
    created_at = Column(DateTime, default=utcnow_naive)

    dispute = relationship("DisputeModel", back_populates="evidences")


class SagaStateModel(Base):
    __tablename__ = "saga_states"

    saga_id = Column(String(36), primary_key=True)
    dispute_id = Column(String(36), nullable=False, index=True)
    booking_id = Column(String(36), nullable=False, index=True)
    saga_type = Column(String(20), nullable=False)  # REFUND, PAYOUT
    current_state = Column(String(50), nullable=False, index=True)
    retry_count = Column(Integer, default=0, nullable=False)
    last_error = Column(Text, nullable=True)
    companion_wallet_id = Column(String(36), nullable=True)
    commission_rate = Column(Float, nullable=True)
    version = Column(Integer, default=1, nullable=False)
    created_at = Column(DateTime, default=utcnow_naive)
    updated_at = Column(DateTime, default=utcnow_naive, onupdate=utcnow_naive)

    __mapper_args__ = {"version_id_col": version}


class OutboxModel(Base):
    __tablename__ = "outbox"

    id = Column(Integer, primary_key=True, autoincrement=True)
    event_id = Column(String(36), unique=True, nullable=False)
    event_type = Column(String(200), nullable=False)
    payload = Column(Text, nullable=False)  # JSON String format
    created_at = Column(DateTime, default=utcnow_naive)
    processed = Column(Boolean, default=False, index=True)


class ProcessedEventModel(Base):
    __tablename__ = "processed_events"

    event_id = Column(String(36), primary_key=True)
    processed_at = Column(DateTime, default=utcnow_naive)
