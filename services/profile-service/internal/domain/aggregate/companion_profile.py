from typing import List, Optional
from internal.domain.vo import Location, MediaUrl
from internal.domain.errors import InvalidProfileStatusTransitionError
from internal.domain.events import (
    DomainEvent,
    ProfileCreated,
    ProfileUpdated,
    ProfileApproved,
    ProfileRejected
)

class CompanionProfile:
    def __init__(
        self,
        companion_id: str,
        user_id: str,
        display_name: str,
        intro_text: str,
        status: str,
        available_cities: List[Location],
        avatar_url: Optional[MediaUrl] = None
    ):
        self.companion_id = companion_id
        self.user_id = user_id
        self.display_name = display_name
        self.intro_text = intro_text
        self.status = status # PENDING, APPROVED, REJECTED
        self.available_cities = available_cities
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
        companion_id: str,
        user_id: str,
        display_name: str,
        intro_text: str,
        available_cities: List[Location]
    ) -> 'CompanionProfile':
        profile = cls(
            companion_id=companion_id,
            user_id=user_id,
            display_name=display_name,
            intro_text=intro_text,
            status="PENDING", # Starts as PENDING for manual admin approval
            available_cities=available_cities
        )
        
        profile.add_event(
            ProfileCreated(
                companion_id=companion_id,
                user_id=user_id,
                display_name=display_name,
                available_cities=[str(city) for city in available_cities]
            )
        )
        return profile

    def update(
        self,
        display_name: str,
        intro_text: str,
        available_cities: List[Location],
        avatar_url: Optional[MediaUrl]
    ):
        self.display_name = display_name
        self.intro_text = intro_text
        self.available_cities = available_cities
        self.avatar_url = avatar_url
        
        self.add_event(
            ProfileUpdated(
                companion_id=self.companion_id,
                display_name=display_name,
                intro_text=intro_text,
                available_cities=[str(city) for city in available_cities]
            )
        )

    def approve(self, admin_id: str):
        if self.status != "PENDING":
            raise InvalidProfileStatusTransitionError(self.status, "APPROVED")
            
        self.status = "APPROVED"
        self.add_event(
            ProfileApproved(
                companion_id=self.companion_id,
                approved_by=admin_id
            )
        )

    def reject(self, admin_id: str, reason: str):
        if self.status != "PENDING":
            raise InvalidProfileStatusTransitionError(self.status, "REJECTED")
            
        self.status = "REJECTED"
        self.add_event(
            ProfileRejected(
                companion_id=self.companion_id,
                rejected_by=admin_id,
                reason=reason
            )
        )
