from typing import Dict, Any, Optional
from internal.domain.errors import (
    ProfileNotFoundError,
    ScenarioNotFoundError,
    ProfileNotAvailableError,
)
from internal.domain.repository import (
    ICompanionProfileRepository,
    IScenarioRepository,
    IMediaAssetRepository,
)


from internal.application.port import IStoragePort


class ProfileQueryService:
    def __init__(
        self,
        profile_repo: ICompanionProfileRepository,
        scenario_repo: IScenarioRepository,
        media_repo: IMediaAssetRepository,
        storage: IStoragePort,
    ):
        self.profile_repo = profile_repo
        self.scenario_repo = scenario_repo
        self.media_repo = media_repo
        self.storage = storage

    async def get_my_profile(self, user_id: str) -> Dict[str, Any]:
        profile = await self.profile_repo.find_by_user_id(user_id)
        if not profile:
            raise ProfileNotFoundError(f"user_id: {user_id}")

        return await self.get_companion_detail(profile.companion_id, public=False)

    async def get_companion_detail(
        self, companion_id: str, public: bool = False
    ) -> Dict[str, Any]:
        profile = await self.profile_repo.find_by_id(companion_id)
        if not profile:
            raise ProfileNotFoundError(companion_id)

        if public and profile.status != "APPROVED":
            raise ProfileNotAvailableError(companion_id)

        # Get scenarios
        scenarios = await self.scenario_repo.find_by_companion_id(companion_id)

        # Get media
        voice_intros = await self.media_repo.find_by_companion_id_and_type(
            companion_id, "VOICE_INTRO"
        )
        albums = await self.media_repo.find_by_companion_id_and_type(
            companion_id, "ALBUM"
        )

        voice_url = None
        if voice_intros:
            raw_url = voice_intros[0].file_url.url
            from urllib.parse import urlparse

            parsed = urlparse(raw_url)
            key = parsed.path.lstrip("/")
            # Generate 5-minute presigned GET URL
            voice_url = self.storage.generate_presigned_get_url(key, expires_in=300)

        album_urls = [asset.file_url.url for asset in albums]

        return {
            "companionId": profile.companion_id,
            "displayName": profile.display_name,
            "avatarUrl": profile.avatar_url.url if profile.avatar_url else None,
            "introText": profile.intro_text,
            "availableCities": [str(city) for city in profile.available_cities],
            "voiceIntroUrl": voice_url,
            "albumUrls": album_urls,
            "averageRating": 4.9,  # Seeded/Mocked for MVP, dynamically updated by review events in Phase 5
            "totalReviews": 150,  # Seeded/Mocked for MVP
            "status": profile.status,
            "scenarios": [
                {
                    "id": sc.scenario_id,
                    "title": sc.title,
                    "description": sc.description,
                    "price": sc.price.amount,
                    "duration": sc.duration_minutes.minutes,
                    "status": sc.status,
                }
                for sc in scenarios
                if sc.status == "ACTIVE" or profile.status == "PENDING"
            ],
        }

    async def get_scenario_snapshot(self, scenario_id: str) -> Dict[str, Any]:
        scenario = await self.scenario_repo.find_by_id(scenario_id)
        if not scenario:
            raise ScenarioNotFoundError(scenario_id)

        return {
            "scenario_id": scenario.scenario_id,
            "companion_id": scenario.companion_id,
            "title": scenario.title,
            "price": scenario.price.amount,
            "duration_minutes": scenario.duration_minutes.minutes,
        }

    async def search_companions(
        self,
        name: Optional[str] = None,
        city: Optional[str] = None,
        min_price: Optional[int] = None,
        max_price: Optional[int] = None,
        page: int = 1,
        page_size: int = 10,
    ) -> Dict[str, Any]:
        offset = (page - 1) * page_size

        # If the repository has a custom search method, use it
        if hasattr(self.profile_repo, "search_approved"):
            companions, total = await self.profile_repo.search_approved(
                name=name,
                city=city,
                min_price=min_price,
                max_price=max_price,
                offset=offset,
                limit=page_size,
            )
        else:
            companions, total = [], 0

        data = []
        for comp in companions:
            # Find starting price (lowest active scenario price)
            scenarios = await self.scenario_repo.find_by_companion_id(comp.companion_id)
            active_scenarios = [s for s in scenarios if s.status == "ACTIVE"]
            starting_price = (
                min([s.price.amount for s in active_scenarios])
                if active_scenarios
                else 0
            )

            data.append(
                {
                    "companionId": comp.companion_id,
                    "displayName": comp.display_name,
                    "avatarUrl": comp.avatar_url.url if comp.avatar_url else None,
                    "averageRating": 4.9,  # Default rating for search cards
                    "city": str(comp.available_cities[0])
                    if comp.available_cities
                    else "Hanoi",
                    "startingPrice": starting_price,
                }
            )

        return {"data": data, "total": total, "page": page, "pageSize": page_size}
