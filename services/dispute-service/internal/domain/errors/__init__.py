class DomainError(Exception):
    """Base domain error exception."""

    pass


class DisputeNotFoundError(DomainError):
    def __init__(self, dispute_id: str):
        super().__init__(f"Dispute not found: {dispute_id}")


class DuplicateOpenDisputeError(DomainError):
    def __init__(self, booking_id: str):
        # [INV-D01] Mỗi BookingId chỉ được phép tồn tại duy nhất 1 Dispute ở trạng thái OPEN hoặc RESOLVING.
        super().__init__(
            f"[INV-D01] Booking {booking_id} already has an active dispute (OPEN or RESOLVING)"
        )


class DisputeAlreadyResolvedError(DomainError):
    def __init__(self, dispute_id: str, current_status: str):
        # [INV-D02] Trạng thái cuối cùng không được phép thay đổi lại.
        super().__init__(
            f"[INV-D02] Dispute {dispute_id} is already resolved with status: {current_status}. "
            f"Terminal states are immutable to prevent SAGA re-execution."
        )


class InvalidDisputeStatusTransitionError(DomainError):
    def __init__(self, current: str, target: str):
        super().__init__(
            f"Invalid dispute status transition from {current} to {target}"
        )


class InvalidDisputeReasonError(DomainError):
    def __init__(self, reason: str, valid_reasons: set):
        super().__init__(
            f"Invalid dispute reason: '{reason}'. Must be one of: {valid_reasons}"
        )


class InvalidResolutionError(DomainError):
    def __init__(self, resolution: str, valid_resolutions: set):
        super().__init__(
            f"Invalid resolution: '{resolution}'. Must be one of: {valid_resolutions}"
        )


class InvalidSagaTransitionError(DomainError):
    def __init__(self, saga_id: str, current_state: str, event: str):
        super().__init__(
            f"Invalid saga transition for {saga_id}: cannot process '{event}' in state '{current_state}'"
        )
