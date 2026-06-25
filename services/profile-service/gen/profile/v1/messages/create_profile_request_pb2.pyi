from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class CreateProfileRequest(_message.Message):
    __slots__ = ("user_id", "display_name", "bio", "available_cities")
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    BIO_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_CITIES_FIELD_NUMBER: _ClassVar[int]
    user_id: str
    display_name: str
    bio: str
    available_cities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, user_id: _Optional[str] = ..., display_name: _Optional[str] = ..., bio: _Optional[str] = ..., available_cities: _Optional[_Iterable[str]] = ...) -> None: ...
