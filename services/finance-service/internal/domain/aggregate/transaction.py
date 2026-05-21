from internal.domain.vo import Money


class Transaction:
    def __init__(
        self,
        transaction_id: str,
        user_id: str,
        amount: Money,
        type: str,  # TOPUP, BOOKING_RESERVATION, ESCROW_RELEASE, PENALTY_DEDUCTION, REFUND
        status: str,  # PENDING, SUCCESS, FAILED
        reference_id: str,
    ):
        self.transaction_id = transaction_id
        self.user_id = user_id
        self.amount = amount
        self.type = type
        self.status = status
        self.reference_id = reference_id

    @classmethod
    def create(
        cls,
        transaction_id: str,
        user_id: str,
        amount: Money,
        type: str,
        status: str,
        reference_id: str,
    ) -> "Transaction":
        return cls(
            transaction_id=transaction_id,
            user_id=user_id,
            amount=amount,
            type=type,
            status=status,
            reference_id=reference_id,
        )

    def success(self):
        self.status = "SUCCESS"

    def fail(self):
        self.status = "FAILED"
