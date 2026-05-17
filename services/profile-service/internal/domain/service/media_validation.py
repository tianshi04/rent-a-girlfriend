from internal.domain.repository import IMediaAssetRepository
from internal.domain.errors import AlbumLimitExceededError


class MediaValidationService:
    def __init__(self, media_repo: IMediaAssetRepository):
        self.media_repo = media_repo

    async def validate_album_limit(self, companion_id: str, max_limit: int = 4):
        # BR-12: Companion được upload 1 avatar + tối đa 4 ảnh album (tổng cộng 5 ảnh)
        # Trong database, chúng ta check số lượng ảnh album hiện tại của companion
        current_albums = await self.media_repo.find_by_companion_id_and_type(
            companion_id=companion_id, asset_type="ALBUM"
        )
        if len(current_albums) >= max_limit:
            raise AlbumLimitExceededError(companion_id, max_limit)
