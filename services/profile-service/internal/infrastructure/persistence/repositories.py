import json
from typing import List, Optional, Tuple
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import func, select
from internal.domain.aggregate import CompanionProfile, Scenario, MediaAsset
from internal.domain.vo import Location, MediaUrl, Money, Duration
from internal.domain.repository import (
    ICompanionProfileRepository,
    IScenarioRepository,
    IMediaAssetRepository,
)
from internal.infrastructure.persistence.models import (
    CompanionProfileModel,
    ScenarioModel,
    MediaAssetModel,
)


class CompanionProfileRepository(ICompanionProfileRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, profile: CompanionProfile) -> None:
        model = CompanionProfileModel(
            companion_id=profile.companion_id,
            user_id=profile.user_id,
            display_name=profile.display_name,
            bio=profile.bio,
            role=profile.role,
            status=profile.status,
            available_cities=json.dumps(
                [str(city) for city in profile.available_cities]
            ),
            avatar_url=profile.avatar_url.url if profile.avatar_url else None,
        )
        await self.session.merge(model)
        await self.session.flush()

    async def find_by_id(self, companion_id: str) -> Optional[CompanionProfile]:
        stmt = select(CompanionProfileModel).filter_by(companion_id=companion_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    async def find_by_user_id(self, user_id: str) -> Optional[CompanionProfile]:
        stmt = select(CompanionProfileModel).filter_by(user_id=user_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    async def search_approved(
        self,
        name: Optional[str],
        city: Optional[str],
        min_price: Optional[int],
        max_price: Optional[int],
        offset: int,
        limit: int,
    ) -> Tuple[List[CompanionProfile], int]:
        # Perform query joining scenarios to filter minimum price
        query = select(CompanionProfileModel).filter(
            CompanionProfileModel.status == "APPROVED",
            CompanionProfileModel.role == "COMPANION"
        )

        if name:
            query = query.filter(CompanionProfileModel.display_name.ilike(f"%{name}%"))

        if city:
            # Check city filter inside JSON array string
            query = query.filter(
                CompanionProfileModel.available_cities.like(f'%"{city}"%')
            )

        # Joined price query
        if min_price is not None or max_price is not None:
            # Filter companion profiles that have at least one active scenario in price range
            subquery = select(ScenarioModel.companion_id).filter(
                ScenarioModel.status == "ACTIVE"
            )
            if min_price is not None:
                subquery = subquery.filter(ScenarioModel.price >= min_price)
            if max_price is not None:
                subquery = subquery.filter(ScenarioModel.price <= max_price)
            subquery = subquery.distinct()
            query = query.filter(CompanionProfileModel.companion_id.in_(subquery))

        # Asynchronously get the count
        count_query = select(func.count()).select_from(query.subquery())
        count_result = await self.session.execute(count_query)
        total = count_result.scalar() or 0

        query = query.offset(offset).limit(limit)
        result = await self.session.execute(query)
        models = result.scalars().all()

        return [self._to_domain(m) for m in models], total

    def _to_domain(self, model: CompanionProfileModel) -> CompanionProfile:
        cities = [
            Location(city_str)
            for city_str in json.loads(model.available_cities or "[]")
        ]
        avatar = MediaUrl(model.avatar_url) if model.avatar_url else None

        profile = CompanionProfile(
            companion_id=model.companion_id,
            user_id=model.user_id,
            display_name=model.display_name,
            bio=model.bio or "",
            status=model.status,
            available_cities=cities,
            role=model.role,
            avatar_url=avatar,
        )
        return profile


class ScenarioRepository(IScenarioRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, scenario: Scenario) -> None:
        model = ScenarioModel(
            scenario_id=scenario.scenario_id,
            companion_id=scenario.companion_id,
            title=scenario.title,
            description=scenario.description,
            price=scenario.price.amount,
            duration_minutes=scenario.duration_minutes.minutes,
            status=scenario.status,
        )
        await self.session.merge(model)
        await self.session.flush()

    async def find_by_id(self, scenario_id: str) -> Optional[Scenario]:
        stmt = select(ScenarioModel).filter_by(scenario_id=scenario_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    async def find_by_companion_id(self, companion_id: str) -> List[Scenario]:
        stmt = select(ScenarioModel).filter_by(companion_id=companion_id)
        result = await self.session.execute(stmt)
        models = result.scalars().all()
        return [self._to_domain(m) for m in models]

    async def count_by_companion_id(self, companion_id: str) -> int:
        stmt = select(func.count(ScenarioModel.scenario_id)).filter_by(
            companion_id=companion_id, status="ACTIVE"
        )
        result = await self.session.execute(stmt)
        return result.scalar() or 0

    async def delete(self, scenario_id: str, companion_id: str) -> None:
        stmt = select(ScenarioModel).filter_by(
            scenario_id=scenario_id, companion_id=companion_id
        )
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if model:
            await self.session.delete(model)
            await self.session.flush()

    def _to_domain(self, model: ScenarioModel) -> Scenario:
        money = Money(model.price)
        duration = Duration(model.duration_minutes)
        return Scenario(
            scenario_id=model.scenario_id,
            companion_id=model.companion_id,
            title=model.title,
            description=model.description or "",
            price=money,
            duration_minutes=duration,
            status=model.status,
        )


class MediaAssetRepository(IMediaAssetRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, asset: MediaAsset) -> None:
        model = MediaAssetModel(
            asset_id=asset.asset_id,
            companion_id=asset.companion_id,
            file_url=asset.file_url.url,
            asset_type=asset.asset_type,
            size_bytes=asset.size_bytes,
            duration_seconds=asset.duration_seconds,
            status=asset.status,
        )
        await self.session.merge(model)
        await self.session.flush()

    async def find_by_id(self, asset_id: str) -> Optional[MediaAsset]:
        stmt = select(MediaAssetModel).filter_by(asset_id=asset_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    async def find_by_companion_id_and_type(
        self, companion_id: str, asset_type: str
    ) -> List[MediaAsset]:
        stmt = select(MediaAssetModel).filter_by(
            companion_id=companion_id, asset_type=asset_type
        )
        result = await self.session.execute(stmt)
        models = result.scalars().all()
        return [self._to_domain(m) for m in models]

    async def delete(self, asset_id: str) -> None:
        stmt = select(MediaAssetModel).filter_by(asset_id=asset_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if model:
            await self.session.delete(model)
            await self.session.flush()

    def _to_domain(self, model: MediaAssetModel) -> MediaAsset:
        url = MediaUrl(model.file_url)
        return MediaAsset(
            asset_id=model.asset_id,
            companion_id=model.companion_id,
            file_url=url,
            asset_type=model.asset_type,
            size_bytes=model.size_bytes,
            duration_seconds=model.duration_seconds,
            status=model.status,
        )
