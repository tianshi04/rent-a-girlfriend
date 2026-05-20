from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class RegisterAlbumImageRequest(_message.Message):
    __slots__ = ("companion_id", "file_url", "size_bytes")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_URL_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    file_url: str
    size_bytes: int
    def __init__(self, companion_id: _Optional[str] = ..., file_url: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...
