from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class UpdateScenarioRequest(_message.Message):
    __slots__ = ("scenario_id", "companion_id", "title", "description", "price", "duration_minutes", "status")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    companion_id: str
    title: str
    description: str
    price: int
    duration_minutes: int
    status: str
    def __init__(self, scenario_id: _Optional[str] = ..., companion_id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., price: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...
