from typing import List, Optional
from internal.domain.vo import MediaUrl
from internal.domain.errors import (
    VoiceIntroDurationExceededError,
    VoiceIntroSizeExceededError,
    AlbumImageSizeExceededError,
)
from internal.domain.events import (
    DomainEvent,
    VoiceIntroRegistered,
    AlbumImageRegistered,
)


class MediaAsset:
    def __init__(
        self,
        asset_id: str,
        companion_id: str,
        file_url: MediaUrl,
        asset_type: str,  # VOICE_INTRO, ALBUM
        size_bytes: int,
        duration_seconds: Optional[int] = None,
        status: str = "PENDING",
    ):
        self.asset_id = asset_id
        self.companion_id = companion_id
        self.file_url = file_url
        self.asset_type = asset_type
        self.size_bytes = size_bytes
        self.duration_seconds = duration_seconds
        self.status = status  # PENDING, APPROVED, REJECTED
        self.events: List[DomainEvent] = []

    def add_event(self, event: DomainEvent):
        self.events.append(event)

    def clear_events(self) -> List[DomainEvent]:
        events = self.events
        self.events = []
        return events

    @classmethod
    def create_voice_intro(
        cls,
        asset_id: str,
        companion_id: str,
        file_url: MediaUrl,
        size_bytes: int,
        duration_seconds: int,
    ) -> "MediaAsset":
        # [INV-P04] Nếu AssetType là VOICE, DurationSec không được vượt quá 30 giây
        if duration_seconds > 30:
            raise VoiceIntroDurationExceededError(duration_seconds)

        # [INV-P05] Nếu AssetType là VOICE, SizeBytes không được vượt quá 5MB (5 * 1024 * 1024)
        if size_bytes > 5 * 1024 * 1024:
            raise VoiceIntroSizeExceededError(size_bytes)

        media = cls(
            asset_id=asset_id,
            companion_id=companion_id,
            file_url=file_url,
            asset_type="VOICE_INTRO",
            size_bytes=size_bytes,
            duration_seconds=duration_seconds,
            status="APPROVED",  # Approved if size/duration invariants are satisfied!
        )

        media.add_event(
            VoiceIntroRegistered(
                companion_id=companion_id,
                asset_id=asset_id,
                file_url=file_url.url,
                duration_seconds=duration_seconds,
                size_bytes=size_bytes,
            )
        )
        return media

    @classmethod
    def create_album_image(
        cls, asset_id: str, companion_id: str, file_url: MediaUrl, size_bytes: int
    ) -> "MediaAsset":
        # BR-12: Dung lượng ảnh album không được vượt quá 2MB (2 * 1024 * 1024)
        if size_bytes > 2 * 1024 * 1024:
            raise AlbumImageSizeExceededError(size_bytes)

        media = cls(
            asset_id=asset_id,
            companion_id=companion_id,
            file_url=file_url,
            asset_type="ALBUM",
            size_bytes=size_bytes,
            status="APPROVED",
        )

        media.add_event(
            AlbumImageRegistered(
                companion_id=companion_id,
                asset_id=asset_id,
                file_url=file_url.url,
                size_bytes=size_bytes,
            )
        )
        return media
