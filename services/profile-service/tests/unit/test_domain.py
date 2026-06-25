import pytest
from internal.domain.vo import Money, Duration, Location, MediaUrl
from internal.domain.aggregate import UserProfile, CompanionProfile, Scenario, MediaAsset
from internal.domain.errors import (
    VoiceIntroDurationExceededError,
    VoiceIntroSizeExceededError,
    AlbumImageSizeExceededError,
)

# --- Value Objects Testing ---


def test_money_valid():
    m = Money(100)
    assert m.amount == 100
    assert str(m) == "100 Kano-Coins"


def test_money_invalid():
    with pytest.raises(ValueError, match=r"\[INV-P01\] Price must be greater than 0"):
        Money(0)
    with pytest.raises(ValueError, match=r"\[INV-P01\] Price must be greater than 0"):
        Money(-50)
    with pytest.raises(TypeError):
        Money("100")


def test_duration_valid():
    d = Duration(60)
    assert d.minutes == 60
    assert str(d) == "60 minutes"


def test_duration_invalid():
    with pytest.raises(
        ValueError, match=r"\[INV-P02\] Duration must be 60, 120, or 180 minutes"
    ):
        Duration(30)
    with pytest.raises(TypeError):
        Duration("60")


def test_location_valid():
    loc = Location("Hanoi")
    assert loc.city == "Hanoi"
    assert str(loc) == "Hanoi"


def test_location_invalid():
    with pytest.raises(ValueError, match="City name cannot be empty"):
        Location(" ")


def test_media_url_valid():
    u = MediaUrl("https://storage.rentgf.com/photo.png")
    assert u.url == "https://storage.rentgf.com/photo.png"


def test_media_url_invalid():
    with pytest.raises(ValueError, match="URL must start with http:// or https://"):
        MediaUrl("ftp://file.com")


# --- UserProfile & CompanionProfile Aggregate Testing ---


def test_user_profile_creation():
    profile = UserProfile.create(
        user_id="user_123",
        display_name="Kano Chizuru",
        bio="Rental Girlfriend of your dreams",
        role="CLIENT",
    )

    assert profile.user_id == "user_123"
    assert profile.role == "CLIENT"
    assert profile.bio == "Rental Girlfriend of your dreams"

    # Check domain event
    events = profile.clear_events()
    assert len(events) == 1
    assert events[0].companion_id == "user_123"
    assert events[0].user_id == "user_123"


def test_companion_profile_creation():
    comp_profile = CompanionProfile.create(
        companion_id="companion_1",
        available_cities=[Location("Hanoi"), Location("HCM")],
        status="APPROVED",
    )

    assert comp_profile.companion_id == "companion_1"
    assert comp_profile.status == "APPROVED"
    assert len(comp_profile.available_cities) == 2


def test_companion_profile_status_transitions():
    profile = CompanionProfile.create(
        companion_id="companion_1",
        available_cities=[Location("Hanoi")],
    )

    # Status is APPROVED by default now
    assert profile.status == "APPROVED"
    profile.approve("admin_id_9")
    assert profile.status == "APPROVED"

    profile.reject("admin_id_9", "Inappropriate")
    assert profile.status == "REJECTED"


def test_user_profile_role_upgrade():
    profile = UserProfile.create(
        user_id="user_123",
        display_name="Kano Chizuru",
        role="CLIENT",
    )
    assert profile.role == "CLIENT"
    profile.upgrade_to_companion()
    assert profile.role == "COMPANION"


# --- Scenario Aggregate Testing ---


def test_scenario_creation():
    scenario = Scenario.create(
        scenario_id="sc_1",
        companion_id="companion_1",
        title="Amusement Park Date",
        description="A romantic walk in the amusement park.",
        price=Money(150),
        duration_minutes=Duration(120),
    )

    assert scenario.scenario_id == "sc_1"
    assert scenario.price.amount == 150
    assert scenario.duration_minutes.minutes == 120

    events = scenario.clear_events()
    assert len(events) == 1
    assert events[0].title == "Amusement Park Date"


# --- MediaAsset Aggregate Testing ---


def test_voice_intro_media_valid():
    url = MediaUrl("https://storage.rentgf.com/voice.mp3")
    # Size 4MB, duration 25s -> valid
    media = MediaAsset.create_voice_intro(
        asset_id="asset_voice",
        companion_id="companion_1",
        file_url=url,
        size_bytes=4 * 1024 * 1024,
        duration_seconds=25,
    )
    assert media.asset_type == "VOICE_INTRO"
    assert media.status == "APPROVED"


def test_voice_intro_media_invalid():
    url = MediaUrl("https://storage.rentgf.com/voice.mp3")

    # Duration exceeds 30s
    with pytest.raises(VoiceIntroDurationExceededError):
        MediaAsset.create_voice_intro(
            asset_id="asset_voice",
            companion_id="companion_1",
            file_url=url,
            size_bytes=1000,
            duration_seconds=35,
        )

    # Size exceeds 5MB
    with pytest.raises(VoiceIntroSizeExceededError):
        MediaAsset.create_voice_intro(
            asset_id="asset_voice",
            companion_id="companion_1",
            file_url=url,
            size_bytes=6 * 1024 * 1024,
            duration_seconds=20,
        )


def test_album_image_media_valid():
    url = MediaUrl("https://storage.rentgf.com/image.png")
    # Size 1.5MB -> valid
    media = MediaAsset.create_album_image(
        asset_id="asset_image",
        companion_id="companion_1",
        file_url=url,
        size_bytes=1500000,
    )
    assert media.asset_type == "ALBUM"
    assert media.status == "APPROVED"


def test_album_image_media_invalid():
    url = MediaUrl("https://storage.rentgf.com/image.png")

    # Size exceeds 2MB
    with pytest.raises(AlbumImageSizeExceededError):
        MediaAsset.create_album_image(
            asset_id="asset_image",
            companion_id="companion_1",
            file_url=url,
            size_bytes=3 * 1024 * 1024,
        )
