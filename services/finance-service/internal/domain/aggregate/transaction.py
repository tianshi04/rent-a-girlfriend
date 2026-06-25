from datetime import datetime
from internal.domain.vo import Money, TransactionType


class Transaction:
    def __init__(
        self,
        transaction_id: str,
        user_id: str,
        amount: Money,
        type: TransactionType,
        status: str,  # PENDING, SUCCESS, FAILED
        reference_id: str,
        created_at: datetime = None,
    ):
        self.transaction_id = transaction_id
        self.user_id = user_id
        self.amount = amount
        self.type = type
        self.status = status
        self.reference_id = reference_id
        self.created_at = created_at

    @classmethod
    def create(
        cls,
        transaction_id: str,
        user_id: str,
        amount: Money,
        type: TransactionType,
        status: str,
        reference_id: str,
        created_at: datetime = None,
    ) -> "Transaction":
        return cls(
            transaction_id=transaction_id,
            user_id=user_id,
            amount=amount,
            type=type,
            status=status,
            reference_id=reference_id,
            created_at=created_at,
        )

    def success(self):
        self.status = "SUCCESS"

    def fail(self):
        self.status = "FAILED"
