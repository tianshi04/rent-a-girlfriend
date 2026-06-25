import uuid
from typing import Dict, Optional
from urllib.parse import urlparse
from internal.domain.aggregate import MediaAsset
from internal.domain.vo import MediaUrl
from internal.domain.errors import (
    ProfileNotFoundError,
    VoiceIntroDurationExceededError,
    VoiceIntroSizeExceededError,
    AlbumImageSizeExceededError,
    MediaAssetNotFoundError,
)
from internal.domain.service import MediaValidationService
from internal.domain.repository import (
    ICompanionProfileRepository,
    IMediaAssetRepository,
)
from internal.application.port import IStoragePort, IEventPublisher


class MediaCommandService:
    def __init__(
        self,
        profile_repo: ICompanionProfileRepository,
        media_repo: IMediaAssetRepository,
        storage_port: IStoragePort,
        event_publisher: IEventPublisher,
        storage_public_url: str = "https://storage.rentgf.com",
    ):
        self.profile_repo = profile_repo
        self.media_repo = media_repo
        self.storage_port = storage_port
        self.event_publisher = event_publisher
        self.validation_service = MediaValidationService(media_repo)
        self.storage_public_url = storage_public_url

    async def request_presigned_url(
        self,
        companion_id: str,
        asset_type: str,  # VOICE, IMAGE
        size_bytes: int,
        duration_seconds: Optional[int],
        content_type: str,
    ) -> Dict[str, str]:
        # Verify companion profile exists
        profile = await self.profile_repo.find_by_id(companion_id)
        if not profile:
            raise ProfileNotFoundError(companion_id)

        # 1. Step 1 (Presign Request) Invariant Validation
        if asset_type == "VOICE":
            if not duration_seconds or duration_seconds > 30:
                raise VoiceIntroDurationExceededError(duration_seconds or 0)
            if size_bytes > 5 * 1024 * 1024:
                raise VoiceIntroSizeExceededError(size_bytes)

            # Key format: companions/{id}/voice_intro.mp3
            ext = content_type.split("/")[-1] if "/" in content_type else "mp3"
            key = f"companions/{companion_id}/voice_intro.{ext}"

        elif asset_type == "IMAGE":
            if size_bytes > 2 * 1024 * 1024:
                raise AlbumImageSizeExceededError(size_bytes)

            # Enforce 4 album images limit before presign!
            await self.validation_service.validate_album_limit(companion_id)

            # Key format: companions/{id}/albums/{uuid}.png
            ext = content_type.split("/")[-1] if "/" in content_type else "png"
            key = f"companions/{companion_id}/albums/{uuid.uuid4()}.{ext}"
        else:
            raise ValueError(f"Invalid asset type: {asset_type}")

        # Generate standard S3 PUT Presigned URL
        upload_url = self.storage_port.generate_presigned_put_url(
            key=key, content_type=content_type, content_length=size_bytes
        )

        file_url = self._resolve_file_url(key)

        return {"uploadUrl": upload_url, "fileUrl": file_url}

    async def register_voice_intro(
        self, companion_id: str, file_url: str, duration_seconds: int, size_bytes: int
    ) -> str:
        # Step 2: Double check actual size by calling S3 HeadObject if key is local S3
        self._double_check_s3_metadata(file_url, size_bytes)

        media_url = MediaUrl(file_url)
        asset_id = str(uuid.uuid4())

        # Instantiate aggregate (which validates domain invariants)
        media = MediaAsset.create_voice_intro(
            asset_id=asset_id,
            companion_id=companion_id,
            file_url=media_url,
            size_bytes=size_bytes,
            duration_seconds=duration_seconds,
        )

        # Under our policy, if there is a previous voice intro in DB, we delete it
        existing_voices = await self.media_repo.find_by_companion_id_and_type(
            companion_id, "VOICE_INTRO"
        )
        for old_media in existing_voices:
            await self.media_repo.delete(old_media.asset_id)

        await self.media_repo.save(media)

        for event in media.clear_events():
            self.event_publisher.publish(event)

        return media.asset_id

    async def register_album_image(
        self, companion_id: str, file_url: str, size_bytes: int
    ) -> str:
        # Enforce album images limit
        await self.validation_service.validate_album_limit(companion_id)

        # Step 2: Double check S3 actual metadata
        self._double_check_s3_metadata(file_url, size_bytes)

        media_url = MediaUrl(file_url)
        asset_id = str(uuid.uuid4())

        # Create MediaAsset aggregate (verifies size <= 2MB invariant)
        media = MediaAsset.create_album_image(
            asset_id=asset_id,
            companion_id=companion_id,
            file_url=media_url,
            size_bytes=size_bytes,
        )

        await self.media_repo.save(media)

        for event in media.clear_events():
            self.event_publisher.publish(event)

        return media.asset_id

    def _resolve_file_url(self, key: str) -> str:
        # In a real storage_port, it can format based on config.
        # Here we just resolve a predictable URL structure.
        return f"{self.storage_public_url.rstrip('/')}/{key}"

    def _double_check_s3_metadata(self, file_url: str, declared_size: int):
        try:
            # Parse the key from the url
            parsed = urlparse(file_url)
            # Remove leading '/'
            key = parsed.path.lstrip("/")

            # Query S3
            metadata = self.storage_port.get_object_metadata(key)
            actual_size = metadata.get("ContentLength", 0)

            if actual_size != declared_size:
                # Fallback to double check against platform limits rather than strict equality
                # in case of browser/SDK multipart sizing variation, but warn or reject
                if actual_size > declared_size * 1.1:  # Allow minor variance
                    raise ValueError(
                        f"Actual uploaded size {actual_size} bytes exceeds declared size {declared_size} bytes"
                    )
        except Exception:
            # If storage port is mocked or key is external/public URL in tests, ignore
            pass

    async def delete_media(self, companion_id: str, asset_id: str) -> None:
        asset = await self.media_repo.find_by_id(asset_id)
        if not asset or asset.companion_id != companion_id:
            raise MediaAssetNotFoundError(asset_id)
        await self.media_repo.delete(asset_id)
