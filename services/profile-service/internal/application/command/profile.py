from typing import List, Optional
from internal.domain.aggregate import CompanionProfile
from internal.domain.vo import Location, MediaUrl
from internal.domain.errors import ProfileAlreadyExistsError, ProfileNotFoundError
from internal.domain.repository import ICompanionProfileRepository
from internal.application.port import IEventPublisher

class ProfileCommandService:
    def __init__(
        self,
        profile_repo: ICompanionProfileRepository,
        event_publisher: IEventPublisher
    ):
        self.profile_repo = profile_repo
        self.event_publisher = event_publisher

    async def create_profile(
        self,
        companion_id: str,
        user_id: str,
        display_name: str,
        intro_text: str,
        available_cities: List[str]
    ) -> str:
        # Check if user already onboarding companion
        existing = await self.profile_repo.find_by_user_id(user_id)
        if existing:
            raise ProfileAlreadyExistsError(user_id)

        cities = [Location(city) for city in available_cities]
        profile = CompanionProfile.create(
            companion_id=companion_id,
            user_id=user_id,
            display_name=display_name,
            intro_text=intro_text,
            available_cities=cities
        )
        
        await self.profile_repo.save(profile)
        
        for event in profile.clear_events():
            self.event_publisher.publish(event)
            
        return profile.companion_id

    async def update_profile(
        self,
        companion_id: str,
        display_name: str,
        intro_text: str,
        available_cities: List[str],
        avatar_url: Optional[str]
    ) -> None:
        profile = await self.profile_repo.find_by_id(companion_id)
        if not profile:
            raise ProfileNotFoundError(companion_id)

        cities = [Location(city) for city in available_cities]
        media_avatar = MediaUrl(avatar_url) if avatar_url else None
        
        profile.update(
            display_name=display_name,
            intro_text=intro_text,
            available_cities=cities,
            avatar_url=media_avatar
        )
        
        await self.profile_repo.save(profile)
        
        for event in profile.clear_events():
            self.event_publisher.publish(event)

    async def approve_profile(self, companion_id: str, admin_id: str) -> None:
        profile = await self.profile_repo.find_by_id(companion_id)
        if not profile:
            raise ProfileNotFoundError(companion_id)
            
        profile.approve(admin_id)
        await self.profile_repo.save(profile)
        
        for event in profile.clear_events():
            self.event_publisher.publish(event)

    async def reject_profile(self, companion_id: str, admin_id: str, reason: str) -> None:
        profile = await self.profile_repo.find_by_id(companion_id)
        if not profile:
            raise ProfileNotFoundError(companion_id)
            
        profile.reject(admin_id, reason)
        await self.profile_repo.save(profile)
        
        for event in profile.clear_events():
            self.event_publisher.publish(event)
