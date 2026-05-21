from typing import List, Tuple
from internal.domain.vo import Money
from internal.domain.errors import InvalidEscrowStatusTransitionError
from internal.domain.events import (
    DomainEvent,
    EscrowCreated,
    PayoutProcessed,
    EscrowRefunded,
)


class Escrow:
    def __init__(
        self,
        escrow_id: str,
        booking_id: str,
        amount: Money,
        status: str,  # HELD, PAID_OUT, REFUNDED
    ):
        self.escrow_id = escrow_id
        self.booking_id = booking_id
        self.amount = amount
        self.status = status
        self.events: List[DomainEvent] = []

    def add_event(self, event: DomainEvent):
        self.events.append(event)

    def clear_events(self) -> List[DomainEvent]:
        events = self.events
        self.events = []
        return events

    @classmethod
    def create(cls, escrow_id: str, booking_id: str, amount: Money) -> "Escrow":
        """
        [INV-F04] Initial state of Escrow is HELD.
        """
        escrow = cls(
            escrow_id=escrow_id,
            booking_id=booking_id,
            amount=amount,
            status="HELD",
        )
        escrow.add_event(
            EscrowCreated(
                booking_id=booking_id,
                amount=amount.amount,
            )
        )
        return escrow

    def payout(self, companion_id: str, commission_rate: float) -> Tuple[int, int]:
        """
        [INV-F05] Only HELD escrows can be paid out.
        """
        if self.status != "HELD":
            raise InvalidEscrowStatusTransitionError(self.status, "PAID_OUT")

        # Commission Calculation using standard round()
        commission_amount = int(round(self.amount.amount * commission_rate))
        net_amount = self.amount.amount - commission_amount

        self.status = "PAID_OUT"
        self.add_event(
            PayoutProcessed(
                booking_id=self.booking_id,
                companion_id=companion_id,
                amount=self.amount.amount,
                commission_amount=commission_amount,
                net_amount=net_amount,
            )
        )
        return commission_amount, net_amount

    def refund(self, client_id: str, refund_amount: Money) -> None:
        """
        [INV-F05] Only HELD escrows can be refunded.
        """
        if self.status != "HELD":
            raise InvalidEscrowStatusTransitionError(self.status, "REFUNDED")

        self.status = "REFUNDED"
        self.add_event(
            EscrowRefunded(
                booking_id=self.booking_id,
                client_id=client_id,
                refund_amount=refund_amount.amount,
            )
        )
