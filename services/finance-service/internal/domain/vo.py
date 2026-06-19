from dataclasses import dataclass
from enum import Enum
from internal.domain.errors import InvalidAmountError


class TransactionType(str, Enum):
    UNSPECIFIED = "UNSPECIFIED"
    TOPUP = "TOPUP"
    BOOKING_RESERVATION = "BOOKING_RESERVATION"
    ESCROW_RELEASE = "ESCROW_RELEASE"
    PENALTY_DEDUCTION = "PENALTY_DEDUCTION"
    REFUND = "REFUND"



@dataclass(frozen=True)
class Money:
    amount: int  # Kano-Coin is represented as integer

    def __post_init__(self):
        # We allow Money(0) as valid
        if self.amount < 0:
            raise InvalidAmountError(self.amount)

    @classmethod
    def zero(cls) -> "Money":
        return cls(0)

    def add(self, other: "Money") -> "Money":
        return Money(self.amount + other.amount)

    def subtract(self, other: "Money") -> "Money":
        if self.amount < other.amount:
            raise InvalidAmountError(self.amount - other.amount)
        return Money(self.amount - other.amount)

    def __str__(self) -> str:
        return f"{self.amount} Kano-Coin"
