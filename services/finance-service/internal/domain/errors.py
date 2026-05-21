class DomainError(Exception):
    """Base domain exception"""

    pass


class InsufficientBalanceError(DomainError):
    def __init__(self, wallet_id: str, required: int, available: int):
        super().__init__(
            f"[INV-F02] Insufficient available balance in wallet {wallet_id}. "
            f"Required: {required}, Available: {available}."
        )


class InsufficientFrozenBalanceError(DomainError):
    def __init__(self, wallet_id: str, required: int, frozen: int):
        super().__init__(
            f"[INV-F03] Insufficient frozen balance in wallet {wallet_id}. "
            f"Required: {required}, Frozen: {frozen}."
        )


class InvalidAmountError(DomainError):
    def __init__(self, amount: int):
        super().__init__(f"Amount must be greater than zero. Got: {amount}.")


class WalletNotFoundError(DomainError):
    def __init__(self, identifier: str):
        super().__init__(f"Wallet not found for {identifier}.")


class WalletAlreadyExistsError(DomainError):
    def __init__(self, user_id: str):
        super().__init__(f"Wallet already exists for user {user_id}.")


class EscrowNotFoundError(DomainError):
    def __init__(self, booking_id: str):
        super().__init__(f"Escrow not found for booking {booking_id}.")


class EscrowAlreadyExistsError(DomainError):
    def __init__(self, booking_id: str):
        super().__init__(
            f"[INV-F04] An active Escrow already exists for booking {booking_id}."
        )


class InvalidEscrowStatusTransitionError(DomainError):
    def __init__(self, current_status: str, target_status: str):
        super().__init__(
            f"[INV-F05] Cannot transition escrow from {current_status} to {target_status}. "
            f"Only HELD escrows can be processed."
        )
