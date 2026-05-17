from datetime import datetime, timezone
from sqlalchemy import Column, String, Integer, Text, Boolean, DateTime, ForeignKey
from sqlalchemy.orm import declarative_base, relationship

Base = declarative_base()


def utcnow_naive():
    """
    Returns a timezone-naive UTC datetime.
    Required because standard Column(DateTime) columns are TIMESTAMP WITHOUT TIME ZONE,
    and asyncpg strictly prohibits mixing offset-naive and offset-aware datetimes.
    """
    return datetime.now(timezone.utc).replace(tzinfo=None)


class CompanionProfileModel(Base):
    __tablename__ = "companion_profiles"

    companion_id = Column(String(36), primary_key=True)
    user_id = Column(String(36), unique=True, nullable=False, index=True)
    display_name = Column(String(100), nullable=False, index=True)
    intro_text = Column(Text, nullable=False)
    status = Column(String(20), default="PENDING", nullable=False, index=True)
    available_cities = Column(Text, nullable=False)  # list of strings in JSON format
    avatar_url = Column(String(500), nullable=True)
    created_at = Column(DateTime, default=utcnow_naive)
    updated_at = Column(DateTime, default=utcnow_naive, onupdate=utcnow_naive)

    scenarios = relationship(
        "ScenarioModel", back_populates="companion", cascade="all, delete-orphan"
    )
    media_assets = relationship(
        "MediaAssetModel", back_populates="companion", cascade="all, delete-orphan"
    )


class ScenarioModel(Base):
    __tablename__ = "scenarios"

    scenario_id = Column(String(36), primary_key=True)
    companion_id = Column(
        String(36),
        ForeignKey("companion_profiles.companion_id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    title = Column(String(200), nullable=False)
    description = Column(Text, nullable=True)
    price = Column(Integer, nullable=False, index=True)  # Kano-Coin amount
    duration_minutes = Column(Integer, nullable=False)
    status = Column(String(20), default="ACTIVE", nullable=False, index=True)
    created_at = Column(DateTime, default=utcnow_naive)

    companion = relationship("CompanionProfileModel", back_populates="scenarios")


class MediaAssetModel(Base):
    __tablename__ = "media_assets"

    asset_id = Column(String(36), primary_key=True)
    companion_id = Column(
        String(36),
        ForeignKey("companion_profiles.companion_id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    file_url = Column(String(500), nullable=False)
    asset_type = Column(String(30), nullable=False, index=True)  # VOICE_INTRO, ALBUM
    size_bytes = Column(Integer, nullable=False)
    duration_seconds = Column(Integer, nullable=True)
    status = Column(String(20), default="PENDING", nullable=False)
    created_at = Column(DateTime, default=utcnow_naive)

    companion = relationship("CompanionProfileModel", back_populates="media_assets")


class OutboxModel(Base):
    __tablename__ = "outbox"

    id = Column(Integer, primary_key=True, autoincrement=True)
    event_id = Column(String(36), unique=True, nullable=False)
    event_type = Column(String(200), nullable=False)
    payload = Column(Text, nullable=False)  # JSON String format
    created_at = Column(DateTime, default=utcnow_naive)
    processed = Column(Boolean, default=False, index=True)
