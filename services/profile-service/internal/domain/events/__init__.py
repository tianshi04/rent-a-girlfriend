from dataclasses import dataclass, field
from datetime import datetime, timezone
import uuid
from typing import List, Optional

@dataclass(frozen=True, kw_only=True)
class DomainEvent:
    event_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    occurred_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))

@dataclass(frozen=True, kw_only=True)
class ProfileCreated(DomainEvent):
    companion_id: str
    user_id: str
    display_name: str
    available_cities: List[str]

@dataclass(frozen=True, kw_only=True)
class ProfileUpdated(DomainEvent):
    companion_id: str
    display_name: str
    intro_text: str
    available_cities: List[str]

@dataclass(frozen=True, kw_only=True)
class ProfileApproved(DomainEvent):
    companion_id: str
    approved_by: str

@dataclass(frozen=True, kw_only=True)
class ProfileRejected(DomainEvent):
    companion_id: str
    rejected_by: str
    reason: str

@dataclass(frozen=True, kw_only=True)
class ScenarioCreated(DomainEvent):
    scenario_id: str
    companion_id: str
    title: str
    price: int
    duration_minutes: int

@dataclass(frozen=True, kw_only=True)
class ScenarioUpdated(DomainEvent):
    scenario_id: str
    companion_id: str
    title: str
    price: int
    duration_minutes: int
    status: str

@dataclass(frozen=True, kw_only=True)
class ScenarioDeleted(DomainEvent):
    scenario_id: str
    companion_id: str

@dataclass(frozen=True, kw_only=True)
class VoiceIntroUploaded(DomainEvent):
    companion_id: str
    asset_id: str
    file_url: str
    duration_seconds: int
    size_bytes: int

@dataclass(frozen=True, kw_only=True)
class VoiceIntroRejected(DomainEvent):
    companion_id: str
    file_url: str
    reason: str

@dataclass(frozen=True, kw_only=True)
class AlbumImageUploaded(DomainEvent):
    companion_id: str
    asset_id: str
    file_url: str
    size_bytes: int
