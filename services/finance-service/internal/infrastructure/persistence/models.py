from datetime import datetime, timezone
from sqlalchemy import Column, String, Integer, Boolean, DateTime, Text, Enum
from sqlalchemy.orm import declarative_base
from internal.domain.vo import TransactionType

Base = declarative_base()


def utcnow_naive():
    """Returns a naive UTC datetime for SQLAlchemy/TIMESTAMP compatibility."""
    return datetime.now(timezone.utc).replace(tzinfo=None)


class WalletModel(Base):
    __tablename__ = "wallets"

    wallet_id = Column(String(36), primary_key=True)
    user_id = Column(String(36), unique=True, nullable=False, index=True)
    available_balance = Column(Integer, default=0, nullable=False)
    frozen_balance = Column(Integer, default=0, nullable=False)
    created_at = Column(DateTime, default=utcnow_naive)
    updated_at = Column(DateTime, default=utcnow_naive, onupdate=utcnow_naive)


class EscrowModel(Base):
    __tablename__ = "escrows"

    escrow_id = Column(String(36), primary_key=True)
    booking_id = Column(String(36), unique=True, nullable=False, index=True)
    amount = Column(Integer, nullable=False)
    status = Column(String(20), default="HELD", nullable=False, index=True)
    created_at = Column(DateTime, default=utcnow_naive)
    updated_at = Column(DateTime, default=utcnow_naive, onupdate=utcnow_naive)


class TransactionModel(Base):
    __tablename__ = "transactions"

    transaction_id = Column(String(36), primary_key=True)
    user_id = Column(String(36), nullable=False, index=True)
    amount = Column(Integer, nullable=False)
    type = Column(Enum(TransactionType), nullable=False, index=True)
    status = Column(String(20), nullable=False, index=True)  # PENDING, SUCCESS, FAILED
    reference_id = Column(String(36), nullable=False, index=True)
    created_at = Column(DateTime, default=utcnow_naive)


class OutboxModel(Base):
    __tablename__ = "outbox"

    id = Column(Integer, primary_key=True, autoincrement=True)
    event_id = Column(String(36), unique=True, nullable=False)
    event_type = Column(String(200), nullable=False)
    payload = Column(Text, nullable=False)  # JSON string
    created_at = Column(DateTime, default=utcnow_naive)
    processed = Column(Boolean, default=False, index=True)
