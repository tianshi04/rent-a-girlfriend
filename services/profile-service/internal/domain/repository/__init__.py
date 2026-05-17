from abc import ABC, abstractmethod
from typing import List, Optional
from internal.domain.aggregate import CompanionProfile, Scenario, MediaAsset

class ICompanionProfileRepository(ABC):
    @abstractmethod
    async def save(self, profile: CompanionProfile) -> None:
        pass

    @abstractmethod
    async def find_by_id(self, companion_id: str) -> Optional[CompanionProfile]:
        pass

    @abstractmethod
    async def find_by_user_id(self, user_id: str) -> Optional[CompanionProfile]:
        pass

    @abstractmethod
    async def search_approved(
        self,
        name: Optional[str],
        city: Optional[str],
        min_price: Optional[int],
        max_price: Optional[int],
        offset: int,
        limit: int
    ) -> tuple[List[CompanionProfile], int]:
        pass


class IScenarioRepository(ABC):
    @abstractmethod
    async def save(self, scenario: Scenario) -> None:
        pass

    @abstractmethod
    async def find_by_id(self, scenario_id: str) -> Optional[Scenario]:
        pass

    @abstractmethod
    async def find_by_companion_id(self, companion_id: str) -> List[Scenario]:
        pass

    @abstractmethod
    async def count_by_companion_id(self, companion_id: str) -> int:
        pass

    @abstractmethod
    async def delete(self, scenario_id: str, companion_id: str) -> None:
        pass


class IMediaAssetRepository(ABC):
    @abstractmethod
    async def save(self, asset: MediaAsset) -> None:
        pass

    @abstractmethod
    async def find_by_id(self, asset_id: str) -> Optional[MediaAsset]:
        pass

    @abstractmethod
    async def find_by_companion_id_and_type(self, companion_id: str, asset_type: str) -> List[MediaAsset]:
        pass

    @abstractmethod
    async def delete(self, asset_id: str) -> None:
        pass
