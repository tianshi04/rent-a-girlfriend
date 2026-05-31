from dataclasses import dataclass


@dataclass(frozen=True, kw_only=True)
class DomainEvent:
    pass


# --- Dispute Lifecycle Events ---


@dataclass(frozen=True, kw_only=True)
class ReportCreated(DomainEvent):
    """Emitted when a user files a report/dispute against a booking."""

    dispute_id: str
    booking_id: str
    reporter_id: str
    accused_id: str
    reason: str


@dataclass(frozen=True, kw_only=True)
class DisputeAssigned(DomainEvent):
    """Emitted when an admin is assigned to handle a dispute."""

    dispute_id: str
    admin_id: str


@dataclass(frozen=True, kw_only=True)
class DisputeResolvedRefund(DomainEvent):
    """Emitted when admin resolves dispute in favor of client (refund)."""

    dispute_id: str
    booking_id: str
    admin_id: str


@dataclass(frozen=True, kw_only=True)
class DisputeResolvedPayout(DomainEvent):
    """Emitted when admin resolves dispute in favor of companion (payout)."""

    dispute_id: str
    booking_id: str
    admin_id: str


@dataclass(frozen=True, kw_only=True)
class DisputeRejected(DomainEvent):
    """Emitted when admin rejects the dispute (no action taken on escrow)."""

    dispute_id: str
    booking_id: str
    admin_id: str


# --- SAGA Lifecycle Events ---


@dataclass(frozen=True, kw_only=True)
class SagaStepCompleted(DomainEvent):
    """Emitted when a SAGA step completes successfully."""

    saga_id: str
    dispute_id: str
    step_name: str


@dataclass(frozen=True, kw_only=True)
class SagaCompleted(DomainEvent):
    """Emitted when the entire SAGA completes successfully."""

    saga_id: str
    dispute_id: str
    saga_type: str  # REFUND or PAYOUT


@dataclass(frozen=True, kw_only=True)
class SagaStepFailed(DomainEvent):
    """Emitted when a SAGA step fails and needs retry."""

    saga_id: str
    dispute_id: str
    step_name: str
    error: str
    retry_count: int
