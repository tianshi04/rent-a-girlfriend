from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class CreateProfileRequest(_message.Message):
    __slots__ = ("user_id", "display_name", "intro_text", "available_cities")
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    INTRO_TEXT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_CITIES_FIELD_NUMBER: _ClassVar[int]
    user_id: str
    display_name: str
    intro_text: str
    available_cities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, user_id: _Optional[str] = ..., display_name: _Optional[str] = ..., intro_text: _Optional[str] = ..., available_cities: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateProfileRequest(_message.Message):
    __slots__ = ("companion_id", "display_name", "intro_text", "available_cities", "avatar_url")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    INTRO_TEXT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_CITIES_FIELD_NUMBER: _ClassVar[int]
    AVATAR_URL_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    display_name: str
    intro_text: str
    available_cities: _containers.RepeatedScalarFieldContainer[str]
    avatar_url: str
    def __init__(self, companion_id: _Optional[str] = ..., display_name: _Optional[str] = ..., intro_text: _Optional[str] = ..., available_cities: _Optional[_Iterable[str]] = ..., avatar_url: _Optional[str] = ...) -> None: ...

class ApproveProfileRequest(_message.Message):
    __slots__ = ("companion_id", "admin_id")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    ADMIN_ID_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    admin_id: str
    def __init__(self, companion_id: _Optional[str] = ..., admin_id: _Optional[str] = ...) -> None: ...

class RejectProfileRequest(_message.Message):
    __slots__ = ("companion_id", "admin_id", "reason")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    ADMIN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    admin_id: str
    reason: str
    def __init__(self, companion_id: _Optional[str] = ..., admin_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class CreateScenarioRequest(_message.Message):
    __slots__ = ("companion_id", "title", "description", "price", "duration_minutes")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    title: str
    description: str
    price: int
    duration_minutes: int
    def __init__(self, companion_id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., price: _Optional[int] = ..., duration_minutes: _Optional[int] = ...) -> None: ...

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

class DeleteScenarioRequest(_message.Message):
    __slots__ = ("scenario_id", "companion_id")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    companion_id: str
    def __init__(self, scenario_id: _Optional[str] = ..., companion_id: _Optional[str] = ...) -> None: ...

class RegisterVoiceIntroRequest(_message.Message):
    __slots__ = ("companion_id", "file_url", "duration_seconds", "size_bytes")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_URL_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    file_url: str
    duration_seconds: int
    size_bytes: int
    def __init__(self, companion_id: _Optional[str] = ..., file_url: _Optional[str] = ..., duration_seconds: _Optional[int] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class RegisterAlbumImageRequest(_message.Message):
    __slots__ = ("companion_id", "file_url", "size_bytes")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_URL_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    file_url: str
    size_bytes: int
    def __init__(self, companion_id: _Optional[str] = ..., file_url: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class ProfileCommandResponse(_message.Message):
    __slots__ = ("companion_id", "status", "message")
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    companion_id: str
    status: str
    message: str
    def __init__(self, companion_id: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ScenarioCommandResponse(_message.Message):
    __slots__ = ("scenario_id", "status", "message")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    status: str
    message: str
    def __init__(self, scenario_id: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class MediaCommandResponse(_message.Message):
    __slots__ = ("asset_id", "status", "message")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    status: str
    message: str
    def __init__(self, asset_id: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class GetScenarioSnapshotRequest(_message.Message):
    __slots__ = ("scenario_id",)
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    def __init__(self, scenario_id: _Optional[str] = ...) -> None: ...

class ScenarioSnapshotResponse(_message.Message):
    __slots__ = ("scenario_id", "companion_id", "title", "price", "duration_minutes")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    COMPANION_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    companion_id: str
    title: str
    price: int
    duration_minutes: int
    def __init__(self, scenario_id: _Optional[str] = ..., companion_id: _Optional[str] = ..., title: _Optional[str] = ..., price: _Optional[int] = ..., duration_minutes: _Optional[int] = ...) -> None: ...
