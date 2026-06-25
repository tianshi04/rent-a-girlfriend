from typing import List
from internal.domain.vo import Location
from internal.domain.events import (
    DomainEvent,
    ProfileApproved,
    ProfileRejected,
)


class CompanionProfile:
    def __init__(
        self,
        companion_id: str,
        status: str,
        available_cities: List[Location],
    ):
        self.companion_id = companion_id
        self.status = status  # APPROVED
        self.available_cities = available_cities
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
        available_cities: List[Location],
        status: str = "APPROVED",
    ) -> "CompanionProfile":
        profile = cls(
            companion_id=companion_id,
            status=status,
            available_cities=available_cities,
        )
        return profile

    def update(
        self,
        available_cities: List[Location],
    ):
        self.available_cities = available_cities

    def approve(self, admin_id: str):
        self.status = "APPROVED"
        self.add_event(
            ProfileApproved(companion_id=self.companion_id, approved_by=admin_id)
        )

    def reject(self, admin_id: str, reason: str):
        self.status = "REJECTED"
        self.add_event(
            ProfileRejected(
                companion_id=self.companion_id, rejected_by=admin_id, reason=reason
            )
        )
