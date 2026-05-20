from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class DeleteScenarioRequest(_message.Message):
    __slots__ = ("scenario_id", "companion_id")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    companion_id: str
    def __init__(self, scenario_id: _Optional[str] = ..., companion_id: _Optional[str] = ...) -> None: ...
