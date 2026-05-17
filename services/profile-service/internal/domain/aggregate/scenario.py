from typing import List
from internal.domain.vo import Money, Duration
from internal.domain.events import (
    DomainEvent,
    ScenarioCreated,
    ScenarioUpdated,
    ScenarioDeleted
)

class Scenario:
    def __init__(
        self,
        scenario_id: str,
        companion_id: str,
        title: str,
        description: str,
        price: Money,
        duration_minutes: Duration,
        status: str = "ACTIVE"
    ):
        self.scenario_id = scenario_id
        self.companion_id = companion_id
        self.title = title
        self.description = description
        self.price = price
        self.duration_minutes = duration_minutes
        self.status = status # ACTIVE, INACTIVE
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
        scenario_id: str,
        companion_id: str,
        title: str,
        description: str,
        price: Money,
        duration_minutes: Duration
    ) -> 'Scenario':
        scenario = cls(
            scenario_id=scenario_id,
            companion_id=companion_id,
            title=title,
            description=description,
            price=price,
            duration_minutes=duration_minutes,
            status="ACTIVE"
        )
        
        scenario.add_event(
            ScenarioCreated(
                scenario_id=scenario_id,
                companion_id=companion_id,
                title=title,
                price=price.amount,
                duration_minutes=duration_minutes.minutes
            )
        )
        return scenario

    def update(
        self,
        title: str,
        description: str,
        price: Money,
        duration_minutes: Duration,
        status: str
    ):
        self.title = title
        self.description = description
        self.price = price
        self.duration_minutes = duration_minutes
        self.status = status
        
        self.add_event(
            ScenarioUpdated(
                scenario_id=self.scenario_id,
                companion_id=self.companion_id,
                title=title,
                price=price.amount,
                duration_minutes=duration_minutes.minutes,
                status=status
            )
        )

    def delete(self):
        self.add_event(
            ScenarioDeleted(
                scenario_id=self.scenario_id,
                companion_id=self.companion_id
            )
        )
