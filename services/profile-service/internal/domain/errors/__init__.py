class DomainError(Exception):
    """Base domain error exception."""

    pass


class ProfileNotFoundError(DomainError):
    def __init__(self, companion_id: str):
        super().__init__(f"Companion profile not found: {companion_id}")


class ProfileAlreadyExistsError(DomainError):
    def __init__(self, user_id: str):
        super().__init__(f"Companion profile already exists for user: {user_id}")


class ScenarioLimitExceededError(DomainError):
    def __init__(self, companion_id: str, limit: int = 5):
        # [INV-P03] Một Companion chỉ được tạo tối đa N Scenarios (VD: 5 kịch bản)
        super().__init__(
            f"[INV-P03] Companion {companion_id} can have at most {limit} active scenarios"
        )


class ScenarioNotFoundError(DomainError):
    def __init__(self, scenario_id: str):
        super().__init__(f"Scenario not found: {scenario_id}")


class InvalidProfileStatusTransitionError(DomainError):
    def __init__(self, current: str, target: str):
        super().__init__(
            f"Invalid profile status transition from {current} to {target}"
        )


class VoiceIntroDurationExceededError(DomainError):
    def __init__(self, duration: int, limit: int = 30):
        # [INV-P04] Nếu AssetType là VOICE, DurationSec không được vượt quá 30 giây
        super().__init__(
            f"[INV-P04] Voice duration {duration}s exceeds platform limit of {limit}s"
        )


class VoiceIntroSizeExceededError(DomainError):
    def __init__(self, size: int, limit_mb: int = 5):
        # [INV-P05] Nếu AssetType là VOICE, SizeBytes không được vượt quá 5MB
        super().__init__(
            f"[INV-P05] Voice size {size} bytes exceeds platform limit of {limit_mb}MB"
        )


class AlbumImageSizeExceededError(DomainError):
    def __init__(self, size: int, limit_mb: int = 2):
        # BR-12: max 2MB per image
        super().__init__(
            f"Album image size {size} bytes exceeds platform limit of {limit_mb}MB"
        )


class AlbumLimitExceededError(DomainError):
    def __init__(self, companion_id: str, limit: int = 4):
        # BR-12: max 4 album photos
        super().__init__(
            f"Companion {companion_id} can have at most {limit} album images"
        )


class ProfileNotAvailableError(DomainError):
    def __init__(self, companion_id: str):
        super().__init__(f"Companion profile not approved or locked: {companion_id}")
