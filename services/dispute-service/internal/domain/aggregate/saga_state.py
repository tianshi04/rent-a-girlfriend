from typing import List, Optional
from internal.domain.errors import InvalidSagaTransitionError
from internal.domain.events import (
    DomainEvent,
    SagaStepCompleted,
    SagaCompleted,
    SagaStepFailed,
)


class DisputeRefundSaga:
    """
    SAGA State Machine for Dispute Refund flow.

    States:
        REFUNDING → HIDING_REVIEW → DISPUTE_RESOLVED_REFUNDED
                                   ↘ (retry on failure, stay in HIDING_REVIEW)

    Đặc thù: Không Rollback Tiền. Nếu bước Interaction lỗi,
    hệ thống dùng Retry vô hạn thay vì hoàn tiền ngược lại.
    """

    VALID_TRANSITIONS = {
        "REFUNDING": {"HIDING_REVIEW", "DISPUTE_FAILED"},
        "HIDING_REVIEW": {"DISPUTE_RESOLVED_REFUNDED"},  # On failure: stays in HIDING_REVIEW with retry
    }

    def __init__(
        self,
        saga_id: str,
        dispute_id: str,
        booking_id: str,
        current_state: str = "REFUNDING",
        retry_count: int = 0,
        last_error: Optional[str] = None,
        version: int = 1,
    ):
        self.saga_id = saga_id
        self.dispute_id = dispute_id
        self.booking_id = booking_id
        self.current_state = current_state
        self.retry_count = retry_count
        self.last_error = last_error
        self.version = version
        self.events: List[DomainEvent] = []

    def add_event(self, event: DomainEvent):
        self.events.append(event)

    def clear_events(self) -> List[DomainEvent]:
        events = self.events
        self.events = []
        return events

    @classmethod
    def create(cls, saga_id: str, dispute_id: str, booking_id: str) -> "DisputeRefundSaga":
        return cls(
            saga_id=saga_id,
            dispute_id=dispute_id,
            booking_id=booking_id,
            current_state="REFUNDING",
        )

    @property
    def is_completed(self) -> bool:
        return self.current_state == "DISPUTE_RESOLVED_REFUNDED"

    @property
    def is_failed(self) -> bool:
        return self.current_state == "DISPUTE_FAILED"

    @property
    def needs_retry(self) -> bool:
        """Check if the saga is stuck in a retriable state with a previous error."""
        return self.current_state == "HIDING_REVIEW" and self.last_error is not None

    def on_refund_success(self):
        """Finance Service returned refund success. Advance to HIDING_REVIEW."""
        if self.current_state != "REFUNDING":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "refund_success")

        self.current_state = "HIDING_REVIEW"
        self.last_error = None
        self.add_event(
            SagaStepCompleted(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                step_name="REFUNDING",
            )
        )

    def on_refund_failed(self, error: str):
        """Finance Service returned refund failure. SAGA fails entirely."""
        if self.current_state != "REFUNDING":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "refund_failed")

        self.current_state = "DISPUTE_FAILED"
        self.last_error = error
        self.add_event(
            SagaStepFailed(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                step_name="REFUNDING",
                error=error,
                retry_count=self.retry_count,
            )
        )

    def on_hide_review_success(self):
        """Interaction Service successfully hid review and locked chat."""
        if self.current_state != "HIDING_REVIEW":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "hide_review_success")

        self.current_state = "DISPUTE_RESOLVED_REFUNDED"
        self.last_error = None
        self.add_event(
            SagaCompleted(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                saga_type="REFUND",
            )
        )

    def on_hide_review_failed(self, error: str):
        """
        Interaction Service failed to hide review / lock chat.
        Stay in HIDING_REVIEW state and increment retry count.
        Uses Infinite Retry strategy — do NOT rollback the refund.
        """
        if self.current_state != "HIDING_REVIEW":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "hide_review_failed")

        # Stay in same state, increment retry
        self.retry_count += 1
        self.last_error = error
        self.add_event(
            SagaStepFailed(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                step_name="HIDING_REVIEW",
                error=error,
                retry_count=self.retry_count,
            )
        )


class DisputePayoutSaga:
    """
    SAGA State Machine for Dispute Payout flow.

    States:
        PAYING_OUT → LOCKING_CHAT → DISPUTE_RESOLVED_PAID_OUT
                                   ↘ (retry on failure, stay in LOCKING_CHAT)

    Same Infinite Retry strategy as RefundSaga for Interaction step.
    """

    VALID_TRANSITIONS = {
        "PAYING_OUT": {"LOCKING_CHAT", "DISPUTE_FAILED"},
        "LOCKING_CHAT": {"DISPUTE_RESOLVED_PAID_OUT"},
    }

    def __init__(
        self,
        saga_id: str,
        dispute_id: str,
        booking_id: str,
        current_state: str = "PAYING_OUT",
        retry_count: int = 0,
        last_error: Optional[str] = None,
        version: int = 1,
    ):
        self.saga_id = saga_id
        self.dispute_id = dispute_id
        self.booking_id = booking_id
        self.current_state = current_state
        self.retry_count = retry_count
        self.last_error = last_error
        self.version = version
        self.events: List[DomainEvent] = []

    def add_event(self, event: DomainEvent):
        self.events.append(event)

    def clear_events(self) -> List[DomainEvent]:
        events = self.events
        self.events = []
        return events

    @classmethod
    def create(cls, saga_id: str, dispute_id: str, booking_id: str) -> "DisputePayoutSaga":
        return cls(
            saga_id=saga_id,
            dispute_id=dispute_id,
            booking_id=booking_id,
            current_state="PAYING_OUT",
        )

    @property
    def is_completed(self) -> bool:
        return self.current_state == "DISPUTE_RESOLVED_PAID_OUT"

    @property
    def is_failed(self) -> bool:
        return self.current_state == "DISPUTE_FAILED"

    @property
    def needs_retry(self) -> bool:
        return self.current_state == "LOCKING_CHAT" and self.last_error is not None

    def on_payout_success(self):
        """Finance Service returned payout success. Advance to LOCKING_CHAT."""
        if self.current_state != "PAYING_OUT":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "payout_success")

        self.current_state = "LOCKING_CHAT"
        self.last_error = None
        self.add_event(
            SagaStepCompleted(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                step_name="PAYING_OUT",
            )
        )

    def on_payout_failed(self, error: str):
        """Finance Service returned payout failure. SAGA fails entirely."""
        if self.current_state != "PAYING_OUT":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "payout_failed")

        self.current_state = "DISPUTE_FAILED"
        self.last_error = error
        self.add_event(
            SagaStepFailed(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                step_name="PAYING_OUT",
                error=error,
                retry_count=self.retry_count,
            )
        )

    def on_lock_chat_success(self):
        """Interaction Service successfully locked chat room."""
        if self.current_state != "LOCKING_CHAT":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "lock_chat_success")

        self.current_state = "DISPUTE_RESOLVED_PAID_OUT"
        self.last_error = None
        self.add_event(
            SagaCompleted(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                saga_type="PAYOUT",
            )
        )

    def on_lock_chat_failed(self, error: str):
        """
        Interaction Service failed to lock chat.
        Stay in LOCKING_CHAT state and increment retry count.
        Uses Infinite Retry strategy — do NOT rollback the payout.
        """
        if self.current_state != "LOCKING_CHAT":
            raise InvalidSagaTransitionError(self.saga_id, self.current_state, "lock_chat_failed")

        self.retry_count += 1
        self.last_error = error
        self.add_event(
            SagaStepFailed(
                saga_id=self.saga_id,
                dispute_id=self.dispute_id,
                step_name="LOCKING_CHAT",
                error=error,
                retry_count=self.retry_count,
            )
        )
