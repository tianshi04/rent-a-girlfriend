import pytest
from internal.domain.aggregate import DisputeRefundSaga, DisputePayoutSaga
from internal.domain.errors import InvalidSagaTransitionError
from internal.domain.events import SagaStepCompleted, SagaCompleted, SagaStepFailed


def test_refund_saga_successful_flow():
    saga = DisputeRefundSaga.create(
        saga_id="saga-refund-1", dispute_id="disp-1", booking_id="book-1"
    )

    assert saga.current_state == "REFUNDING"
    assert saga.is_completed is False

    # 1. Finance refund successful -> HIDING_REVIEW
    saga.on_refund_success()
    assert saga.current_state == "HIDING_REVIEW"
    assert saga.last_error is None
    events = saga.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], SagaStepCompleted)
    assert events[0].step_name == "REFUNDING"

    # 2. Hide review and lock chat successful -> DISPUTE_RESOLVED_REFUNDED (Completed)
    saga.on_hide_review_success()
    assert saga.current_state == "DISPUTE_RESOLVED_REFUNDED"
    assert saga.is_completed is True
    events = saga.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], SagaCompleted)
    assert events[0].saga_type == "REFUND"


def test_refund_saga_failed_refund():
    saga = DisputeRefundSaga.create(
        saga_id="saga-refund-1", dispute_id="disp-1", booking_id="book-1"
    )

    # If finance refund fails, the SAGA fails entirely (money has not left escrow)
    saga.on_refund_failed("Escrow is already empty or wallet is invalid")
    assert saga.current_state == "DISPUTE_FAILED"
    assert saga.is_failed is True
    assert saga.last_error == "Escrow is already empty or wallet is invalid"
    events = saga.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], SagaStepFailed)
    assert events[0].step_name == "REFUNDING"


def test_refund_saga_failed_interaction_retries():
    saga = DisputeRefundSaga.create(
        saga_id="saga-refund-1", dispute_id="disp-1", booking_id="book-1"
    )

    saga.on_refund_success()
    saga.clear_events()

    # If Interaction Service fails, SAGA stays in HIDING_REVIEW and increments retry count (Infinite Retry)
    saga.on_hide_review_failed("Interaction service timeout")
    assert saga.current_state == "HIDING_REVIEW"
    assert saga.retry_count == 1
    assert saga.last_error == "Interaction service timeout"
    assert saga.needs_retry is True

    events = saga.clear_events()
    assert len(events) == 1
    assert isinstance(events[0], SagaStepFailed)
    assert events[0].step_name == "HIDING_REVIEW"

    # Second retry
    saga.on_hide_review_failed("Database deadlock in interaction")
    assert saga.current_state == "HIDING_REVIEW"
    assert saga.retry_count == 2

    # Eventually succeeds
    saga.on_hide_review_success()
    assert saga.current_state == "DISPUTE_RESOLVED_REFUNDED"
    assert saga.is_completed is True


def test_payout_saga_successful_flow():
    saga = DisputePayoutSaga.create(
        saga_id="saga-payout-1",
        dispute_id="disp-1",
        booking_id="book-1",
        companion_wallet_id="wallet-1",
        commission_rate=0.15,
    )

    assert saga.current_state == "PAYING_OUT"

    saga.on_payout_success()
    assert saga.current_state == "LOCKING_CHAT"

    saga.on_lock_chat_success()
    assert saga.current_state == "DISPUTE_RESOLVED_PAID_OUT"
    assert saga.is_completed is True


def test_saga_invalid_transitions():
    saga = DisputeRefundSaga.create(
        saga_id="saga-refund-1", dispute_id="disp-1", booking_id="book-1"
    )

    # Cannot trigger hide_review_success when in REFUNDING state
    with pytest.raises(InvalidSagaTransitionError):
        saga.on_hide_review_success()
