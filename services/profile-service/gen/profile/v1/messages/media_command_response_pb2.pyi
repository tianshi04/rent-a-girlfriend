from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class MediaCommandResponse(_message.Message):
    __slots__ = ("asset_id", "status", "message")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    status: str
    message: str
    def __init__(self, asset_id: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
