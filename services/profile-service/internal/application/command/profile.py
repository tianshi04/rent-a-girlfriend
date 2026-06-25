from typing import List, Optional
from internal.domain.aggregate import UserProfile, CompanionProfile
from internal.domain.vo import Location, MediaUrl
from internal.domain.errors import ProfileAlreadyExistsError, ProfileNotFoundError
from internal.domain.repository import (
    IUserProfileRepository,
    ICompanionProfileRepository,
)
from internal.application.port import IEventPublisher


class ProfileCommandService:
    def __init__(
        self,
        user_profile_repo: IUserProfileRepository,
        profile_repo: ICompanionProfileRepository,
        event_publisher: IEventPublisher,
    ):
        self.user_profile_repo = user_profile_repo
        self.profile_repo = profile_repo
        self.event_publisher = event_publisher

    async def create_profile(
        self,
        companion_id: str,
        user_id: str,
        display_name: str,
        available_cities: List[str],
        bio: str = "",
        role: str = "CLIENT",
    ) -> str:
        # Check if user already onboarding
        existing = await self.user_profile_repo.find_by_id(user_id)
        if existing:
            raise ProfileAlreadyExistsError(user_id)

        # Create UserProfile
        user_profile = UserProfile.create(
            user_id=user_id,
            display_name=display_name,
            bio=bio,
            role=role,
        )
        await self.user_profile_repo.save(user_profile)

        # If role is COMPANION, also create CompanionProfile
        if role == "COMPANION":
            cities = [Location(city) for city in available_cities]
            companion_profile = CompanionProfile.create(
                companion_id=companion_id,
                available_cities=cities,
                status="APPROVED",
            )
            await self.profile_repo.save(companion_profile)

            # Forward events from companion profile
            for event in companion_profile.clear_events():
                self.event_publisher.publish(event)

        # Forward events from user profile (ProfileCreated)
        # Note: if it's a COMPANION, we can populate available_cities from passed cities for backward compatibility
        for event in user_profile.clear_events():
            # If the event is ProfileCreated and it's a COMPANION, we fill in available_cities
            if hasattr(event, "available_cities") and role == "COMPANION":
                object.__setattr__(event, "available_cities", available_cities)
            self.event_publisher.publish(event)

        return user_id

    async def update_profile(
        self,
        companion_id: str,
        display_name: str,
        available_cities: List[str],
        avatar_url: Optional[str],
        bio: str = "",
    ) -> None:
        user_profile = await self.user_profile_repo.find_by_id(companion_id)
        if not user_profile:
            raise ProfileNotFoundError(companion_id)

        media_avatar = MediaUrl(avatar_url) if avatar_url else None
        user_profile.update(
            display_name=display_name,
            bio=bio,
            avatar_url=media_avatar,
            available_cities=available_cities,
        )
        await self.user_profile_repo.save(user_profile)

        if user_profile.role == "COMPANION":
            companion_profile = await self.profile_repo.find_by_id(companion_id)
            if not companion_profile:
                cities = [Location(city) for city in available_cities]
                companion_profile = CompanionProfile.create(
                    companion_id=companion_id,
                    available_cities=cities,
                    status="APPROVED",
                )
            else:
                cities = [Location(city) for city in available_cities]
                companion_profile.update(available_cities=cities)

            await self.profile_repo.save(companion_profile)

            for event in companion_profile.clear_events():
                self.event_publisher.publish(event)

        for event in user_profile.clear_events():
            self.event_publisher.publish(event)

    async def patch_profile(
        self,
        companion_id: str,
        display_name: Optional[str],
        bio: Optional[str],
        available_cities: Optional[List[str]],
        avatar_url: Optional[str],
    ) -> None:
        user_profile = await self.user_profile_repo.find_by_id(companion_id)
        if not user_profile:
            raise ProfileNotFoundError(companion_id)

        # Apply only the fields that were explicitly provided (non-None)
        new_display_name = (
            display_name if display_name is not None else user_profile.display_name
        )
        new_bio = bio if bio is not None else user_profile.bio
        new_avatar = (
            MediaUrl(avatar_url) if avatar_url is not None else user_profile.avatar_url
        )
        new_cities = available_cities if available_cities is not None else None

        user_profile.update(
            display_name=new_display_name,
            bio=new_bio,
            avatar_url=new_avatar,
            available_cities=new_cities,
        )
        await self.user_profile_repo.save(user_profile)

        if user_profile.role == "COMPANION" and available_cities is not None:
            companion_profile = await self.profile_repo.find_by_id(companion_id)
            cities = [Location(city) for city in available_cities]
            if not companion_profile:
                companion_profile = CompanionProfile.create(
                    companion_id=companion_id,
                    available_cities=cities,
                    status="APPROVED",
                )
            else:
                companion_profile.update(available_cities=cities)

            await self.profile_repo.save(companion_profile)

            for event in companion_profile.clear_events():
                self.event_publisher.publish(event)

        for event in user_profile.clear_events():
            self.event_publisher.publish(event)

    async def upgrade_profile_role(self, user_id: str) -> None:
        user_profile = await self.user_profile_repo.find_by_id(user_id)
        if not user_profile:
            raise ProfileNotFoundError(user_id)

        user_profile.upgrade_to_companion()
        await self.user_profile_repo.save(user_profile)

        # Create CompanionProfile if not exists
        companion_profile = await self.profile_repo.find_by_id(user_id)
        if not companion_profile:
            companion_profile = CompanionProfile.create(
                companion_id=user_id,
                available_cities=[],
                status="APPROVED",
            )
            await self.profile_repo.save(companion_profile)

            for event in companion_profile.clear_events():
                self.event_publisher.publish(event)

        for event in user_profile.clear_events():
            self.event_publisher.publish(event)
