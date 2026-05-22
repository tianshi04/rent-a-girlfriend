from dataclasses import dataclass
from internal.domain.errors import InvalidDisputeReasonError, InvalidResolutionError


VALID_REASONS = {"NO_SHOW", "MISCONDUCT", "FRAUD", "OTHER"}
VALID_RESOLUTIONS = {"REFUND_CLIENT", "PAYOUT_COMPANION", "REJECT"}


@dataclass(frozen=True)
class DisputeReason:
    """Value Object representing a validated dispute reason."""

    value: str

    def __post_init__(self):
        if not isinstance(self.value, str) or self.value not in VALID_REASONS:
            raise InvalidDisputeReasonError(self.value, VALID_REASONS)

    def __str__(self) -> str:
        return self.value


@dataclass(frozen=True)
class Resolution:
    """Value Object representing a validated admin resolution decision."""

    value: str

    def __post_init__(self):
        if not isinstance(self.value, str) or self.value not in VALID_RESOLUTIONS:
            raise InvalidResolutionError(self.value, VALID_RESOLUTIONS)

    @property
    def is_refund(self) -> bool:
        return self.value == "REFUND_CLIENT"

    @property
    def is_payout(self) -> bool:
        return self.value == "PAYOUT_COMPANION"

    @property
    def is_reject(self) -> bool:
        return self.value == "REJECT"

    def __str__(self) -> str:
        return self.value
