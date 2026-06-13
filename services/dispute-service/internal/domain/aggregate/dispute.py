from typing import List, Optional
from internal.domain.vo import DisputeReason
from internal.domain.errors import (
    DisputeAlreadyResolvedError,
    InvalidDisputeStatusTransitionError,
)
from internal.domain.events import (
    DomainEvent,
    ReportCreated,
    DisputeAssigned,
    DisputeResolvedRefund,
    DisputeResolvedPayout,
    DisputeRejected,
)


# Terminal states — once reached, dispute status is immutable [INV-D02]
TERMINAL_STATES = {"REFUNDED", "PAID_OUT", "REJECTED"}

# Valid transition map
VALID_TRANSITIONS = {
    "OPEN": {"RESOLVING"},
    "RESOLVING": {"REFUNDED", "PAID_OUT", "REJECTED"},
}


class DisputeEvidence:
    """Entity con: bằng chứng đính kèm (hình ảnh, text giải trình)."""

    def __init__(self, evidence_id: str, evidence_type: str, content: str):
        self.evidence_id = evidence_id
        self.evidence_type = evidence_type  # TEXT, IMAGE
        self.content = content


class Dispute:
    """
    Aggregate Root: Dispute (Khiếu nại / Tranh chấp)

    State Machine:
        OPEN → RESOLVING → REFUNDED | PAID_OUT | REJECTED

    Invariants:
        [INV-D01] Mỗi BookingId chỉ được phép tồn tại duy nhất 1 Dispute
                  ở trạng thái OPEN hoặc RESOLVING. (Enforced at Application layer)
        [INV-D02] Trạng thái cuối cùng (REFUNDED, PAID_OUT, REJECTED) là bất biến.
        [INV-D03] Chỉ Admin mới có quyền thực thi Resolve. (Enforced at Interface layer)
    """

    def __init__(
        self,
        dispute_id: str,
        booking_id: str,
        reporter_id: str,
        accused_id: str,
        reason: DisputeReason,
        status: str,
        admin_id: Optional[str] = None,
        resolution: Optional[str] = None,
        notes: Optional[str] = None,
        evidences: Optional[List[DisputeEvidence]] = None,
        version: int = 1,
    ):
        self.dispute_id = dispute_id
        self.booking_id = booking_id
        self.reporter_id = reporter_id
        self.accused_id = accused_id
        self.reason = reason
        self.status = status
        self.admin_id = admin_id
        self.resolution = resolution
        self.notes = notes
        self.evidences = evidences or []
        self.version = version
        self.events: List[DomainEvent] = []

    def add_event(self, event: DomainEvent):
        self.events.append(event)

    def clear_events(self) -> List[DomainEvent]:
        events = self.events
        self.events = []
        return events

    def _assert_not_terminal(self):
        """[INV-D02] Guard against modifying terminal states."""
        if self.status in TERMINAL_STATES:
            raise DisputeAlreadyResolvedError(self.dispute_id, self.status)

    def _assert_valid_transition(self, target: str):
        """Validate state machine transition is allowed."""
        self._assert_not_terminal()
        allowed = VALID_TRANSITIONS.get(self.status, set())
        if target not in allowed:
            raise InvalidDisputeStatusTransitionError(self.status, target)

    # --- Factory Method ---

    @classmethod
    def create_report(
        cls,
        dispute_id: str,
        booking_id: str,
        reporter_id: str,
        accused_id: str,
        reason: str,
        evidences: Optional[List[DisputeEvidence]] = None,
    ) -> "Dispute":
        """Factory method: tạo một khiếu nại mới với trạng thái OPEN."""
        dispute_reason = DisputeReason(reason)

        dispute = cls(
            dispute_id=dispute_id,
            booking_id=booking_id,
            reporter_id=reporter_id,
            accused_id=accused_id,
            reason=dispute_reason,
            status="OPEN",
            evidences=evidences or [],
        )

        dispute.add_event(
            ReportCreated(
                dispute_id=dispute_id,
                booking_id=booking_id,
                reporter_id=reporter_id,
                accused_id=accused_id,
                reason=str(dispute_reason),
            )
        )
        return dispute

    # --- Command Methods ---

    def assign_admin(self, admin_id: str):
        """Assign an admin to handle this dispute. OPEN → RESOLVING."""
        self._assert_valid_transition("RESOLVING")
        self.status = "RESOLVING"
        self.admin_id = admin_id

        self.add_event(DisputeAssigned(dispute_id=self.dispute_id, admin_id=admin_id))

    def resolve_refund(self, admin_id: str, notes: str = ""):
        """
        Admin resolves dispute: refund client.
        RESOLVING → REFUNDED.
        Triggers DisputeRefundSaga at Application layer.
        """
        self._assert_valid_transition("REFUNDED")
        self.status = "REFUNDED"
        self.admin_id = admin_id
        self.resolution = "REFUND_CLIENT"
        self.notes = notes

        self.add_event(
            DisputeResolvedRefund(
                dispute_id=self.dispute_id,
                booking_id=self.booking_id,
                admin_id=admin_id,
                reporter_id=self.reporter_id,
                accused_id=self.accused_id,
            )
        )

    def resolve_payout(self, admin_id: str, notes: str = ""):
        """
        Admin resolves dispute: payout to companion.
        RESOLVING → PAID_OUT.
        Triggers DisputePayoutSaga at Application layer.
        """
        self._assert_valid_transition("PAID_OUT")
        self.status = "PAID_OUT"
        self.admin_id = admin_id
        self.resolution = "PAYOUT_COMPANION"
        self.notes = notes

        self.add_event(
            DisputeResolvedPayout(
                dispute_id=self.dispute_id,
                booking_id=self.booking_id,
                admin_id=admin_id,
                reporter_id=self.reporter_id,
                accused_id=self.accused_id,
            )
        )

    def reject(self, admin_id: str, notes: str = ""):
        """
        Admin rejects dispute: no financial action taken.
        RESOLVING → REJECTED.
        No SAGA needed — escrow will follow normal completion flow.
        """
        self._assert_valid_transition("REJECTED")
        self.status = "REJECTED"
        self.admin_id = admin_id
        self.resolution = "REJECT"
        self.notes = notes

        self.add_event(
            DisputeRejected(
                dispute_id=self.dispute_id,
                booking_id=self.booking_id,
                admin_id=admin_id,
                reporter_id=self.reporter_id,
                accused_id=self.accused_id,
            )
        )

    def add_evidence(self, evidence: DisputeEvidence):
        """Add evidence to the dispute (only while not in terminal state)."""
        self._assert_not_terminal()
        self.evidences.append(evidence)
