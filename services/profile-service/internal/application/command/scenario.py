import uuid
from internal.domain.aggregate import Scenario
from internal.domain.vo import Money, Duration
from internal.domain.errors import (
    ProfileNotFoundError,
    ScenarioNotFoundError,
    ScenarioLimitExceededError,
)
from internal.domain.repository import ICompanionProfileRepository, IScenarioRepository
from internal.application.port import IEventPublisher


class ScenarioCommandService:
    def __init__(
        self,
        profile_repo: ICompanionProfileRepository,
        scenario_repo: IScenarioRepository,
        event_publisher: IEventPublisher,
    ):
        self.profile_repo = profile_repo
        self.scenario_repo = scenario_repo
        self.event_publisher = event_publisher

    async def create_scenario(
        self,
        companion_id: str,
        title: str,
        description: str,
        price: int,
        duration_minutes: int,
    ) -> str:
        # Verify companion exists
        profile = await self.profile_repo.find_by_id(companion_id)
        if not profile:
            raise ProfileNotFoundError(companion_id)

        # [INV-P03] Check scenario limit (max 5 active scenarios)
        current_count = await self.scenario_repo.count_by_companion_id(companion_id)
        if current_count >= 5:
            raise ScenarioLimitExceededError(companion_id, 5)

        money = Money(price)
        duration = Duration(duration_minutes)
        scenario_id = str(uuid.uuid4())

        scenario = Scenario.create(
            scenario_id=scenario_id,
            companion_id=companion_id,
            title=title,
            description=description,
            price=money,
            duration_minutes=duration,
        )

        await self.scenario_repo.save(scenario)

        for event in scenario.clear_events():
            self.event_publisher.publish(event)

        return scenario_id

    async def update_scenario(
        self,
        scenario_id: str,
        companion_id: str,
        title: str,
        description: str,
        price: int,
        duration_minutes: int,
        status: str,
    ) -> None:
        scenario = await self.scenario_repo.find_by_id(scenario_id)
        if not scenario or scenario.companion_id != companion_id:
            raise ScenarioNotFoundError(scenario_id)

        money = Money(price)
        duration = Duration(duration_minutes)

        scenario.update(
            title=title,
            description=description,
            price=money,
            duration_minutes=duration,
            status=status,
        )

        await self.scenario_repo.save(scenario)

        for event in scenario.clear_events():
            self.event_publisher.publish(event)

    async def delete_scenario(self, scenario_id: str, companion_id: str) -> None:
        scenario = await self.scenario_repo.find_by_id(scenario_id)
        if not scenario or scenario.companion_id != companion_id:
            raise ScenarioNotFoundError(scenario_id)

        scenario.delete()
        await self.scenario_repo.delete(scenario_id, companion_id)

        for event in scenario.clear_events():
            self.event_publisher.publish(event)
