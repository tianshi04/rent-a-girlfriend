import pytest
from internal.domain.aggregate import Dispute, DisputeEvidence
from internal.domain.errors import DisputeAlreadyResolvedError, InvalidDisputeStatusTransitionError
from internal.domain.events import ReportCreated, DisputeAssigned, DisputeResolvedRefund, DisputeResolvedPayout, DisputeRejected


def test_dispute_creation_flow():
    dispute = Dispute.create_report(
        dispute_id="disp-123",
        booking_id="book-456",
        reporter_id="user-client",
        accused_id="user-companion",
        reason="NO_SHOW",
    )

    assert dispute.dispute_id == "disp-123"
    assert dispute.status == "OPEN"
    assert str(dispute.reason) == "NO_SHOW"
    assert len(dispute.evidences) == 0

    events = dispute.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], ReportCreated)
    assert events[0].dispute_id == "disp-123"
    assert events[0].booking_id == "book-456"


def test_dispute_assign_admin():
    dispute = Dispute.create_report(
        dispute_id="disp-123",
        booking_id="book-456",
        reporter_id="user-client",
        accused_id="user-companion",
        reason="NO_SHOW",
    )
    dispute.clear_events()

    dispute.assign_admin("admin-789")
    assert dispute.status == "RESOLVING"
    assert dispute.admin_id == "admin-789"

    events = dispute.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], DisputeAssigned)
    assert events[0].admin_id == "admin-789"


def test_dispute_resolve_refund():
    dispute = Dispute.create_report(
        dispute_id="disp-123",
        booking_id="book-456",
        reporter_id="user-client",
        accused_id="user-companion",
        reason="NO_SHOW",
    )
    dispute.assign_admin("admin-789")
    dispute.clear_events()

    dispute.resolve_refund("admin-789", "Refunding because companion did not show up.")
    assert dispute.status == "REFUNDED"
    assert dispute.resolution == "REFUND_CLIENT"

    events = dispute.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], DisputeResolvedRefund)
    assert events[0].dispute_id == "disp-123"


def test_dispute_resolve_payout():
    dispute = Dispute.create_report(
        dispute_id="disp-123",
        booking_id="book-456",
        reporter_id="user-client",
        accused_id="user-companion",
        reason="NO_SHOW",
    )
    dispute.assign_admin("admin-789")
    dispute.clear_events()

    dispute.resolve_payout("admin-789", "Payout companion, client claim was invalid.")
    assert dispute.status == "PAID_OUT"
    assert dispute.resolution == "PAYOUT_COMPANION"

    events = dispute.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], DisputeResolvedPayout)


def test_dispute_reject():
    dispute = Dispute.create_report(
        dispute_id="disp-123",
        booking_id="book-456",
        reporter_id="user-client",
        accused_id="user-companion",
        reason="NO_SHOW",
    )
    dispute.assign_admin("admin-789")
    dispute.clear_events()

    dispute.reject("admin-789", "Rejected dispute claim.")
    assert dispute.status == "REJECTED"
    assert dispute.resolution == "REJECT"

    events = dispute.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], DisputeRejected)


def test_dispute_terminal_state_invariance():
    dispute = Dispute.create_report(
        dispute_id="disp-123",
        booking_id="book-456",
        reporter_id="user-client",
        accused_id="user-companion",
        reason="NO_SHOW",
    )
    dispute.assign_admin("admin-789")
    dispute.resolve_refund("admin-789", "Refunding")

    # Trying to modify a terminal state [INV-D02]
    with pytest.raises(DisputeAlreadyResolvedError):
        dispute.assign_admin("admin-different")

    with pytest.raises(DisputeAlreadyResolvedError):
        dispute.resolve_payout("admin-789")


def test_dispute_invalid_status_transition():
    dispute = Dispute.create_report(
        dispute_id="disp-123",
        booking_id="book-456",
        reporter_id="user-client",
        accused_id="user-companion",
        reason="NO_SHOW",
    )
    
    # Cannot resolve an OPEN dispute without assigning an admin first (status must be RESOLVING)
    with pytest.raises(InvalidDisputeStatusTransitionError):
        dispute.resolve_refund("admin-789")
