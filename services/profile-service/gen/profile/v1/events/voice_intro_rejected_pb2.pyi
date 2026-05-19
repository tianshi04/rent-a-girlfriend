from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class VoiceIntroRejected(_message.Message):
    __slots__ = ("companion_id", "file_url", "reason")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_URL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    file_url: str
    reason: str
    def __init__(self, companion_id: _Optional[str] = ..., file_url: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
