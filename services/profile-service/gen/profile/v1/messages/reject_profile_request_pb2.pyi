from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class RejectProfileRequest(_message.Message):
    __slots__ = ("companion_id", "admin_id", "reason")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    ADMIN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    admin_id: str
    reason: str
    def __init__(self, companion_id: _Optional[str] = ..., admin_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
