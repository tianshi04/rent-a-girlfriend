from typing import List, Optional
from internal.domain.vo import MediaUrl
from internal.domain.events import (
    DomainEvent,
    ProfileCreated,
    ProfileUpdated,
)


class UserProfile:
    def __init__(
        self,
        user_id: str,
        display_name: str,
        bio: str,
        role: str = "CLIENT",
        avatar_url: Optional[MediaUrl] = None,
    ):
        self.user_id = user_id
        self.display_name = display_name
        self.bio = bio
        self.role = role
        self.avatar_url = avatar_url
        self.events: List[DomainEvent] = []

    def add_event(self, event: DomainEvent):
        self.events.append(event)

    def clear_events(self) -> List[DomainEvent]:
        events = self.events
        self.events = []
        return events

    @classmethod
    def create(
        cls,
        user_id: str,
        display_name: str,
        bio: str = "",
        role: str = "CLIENT",
    ) -> "UserProfile":
        profile = cls(
            user_id=user_id,
            display_name=display_name,
            bio=bio,
            role=role,
        )
        profile.add_event(
            ProfileCreated(
                companion_id=user_id,
                user_id=user_id,
                display_name=display_name,
                available_cities=[],
            )
        )
        return profile

    def update(
        self,
        display_name: str,
        bio: str,
        avatar_url: Optional[MediaUrl] = None,
        available_cities: Optional[List[str]] = None,
    ):
        self.display_name = display_name
        self.bio = bio
        self.avatar_url = avatar_url

        self.add_event(
            ProfileUpdated(
                companion_id=self.user_id,
                display_name=display_name,
                bio=bio,
                available_cities=available_cities
                if available_cities is not None
                else [],
            )
        )

    def upgrade_to_companion(self):
        self.role = "COMPANION"
