from internal.domain import events as domain_events
from gen.profile.v1.events import (
    profile_created_pb2,
    profile_updated_pb2,
    profile_approved_pb2,
    profile_rejected_pb2,
    scenario_created_pb2,
    scenario_updated_pb2,
    scenario_deleted_pb2,
    voice_intro_registered_pb2,
    voice_intro_rejected_pb2,
    album_image_registered_pb2,
)


class EventMapper:
    @staticmethod
    def to_protobuf(domain_event: domain_events.DomainEvent):
        """
        Maps a domain event (dataclass) to its corresponding protobuf message.
        """
        if isinstance(domain_event, domain_events.ProfileCreated):
            return profile_created_pb2.ProfileCreated(
                companion_id=domain_event.companion_id,
                display_name=domain_event.display_name,
                intro_text=getattr(domain_event, "intro_text", ""),
                available_cities=domain_event.available_cities,
            )

        elif isinstance(domain_event, domain_events.ProfileUpdated):
            return profile_updated_pb2.ProfileUpdated(
                companion_id=domain_event.companion_id,
                display_name=domain_event.display_name,
                intro_text=domain_event.intro_text,
                available_cities=domain_event.available_cities,
                avatar_url=getattr(domain_event, "avatar_url", ""),
            )

        elif isinstance(domain_event, domain_events.ProfileApproved):
            return profile_approved_pb2.ProfileApproved(
                companion_id=domain_event.companion_id,
                admin_id=domain_event.approved_by,
            )

        elif isinstance(domain_event, domain_events.ProfileRejected):
            return profile_rejected_pb2.ProfileRejected(
                companion_id=domain_event.companion_id,
                admin_id=domain_event.rejected_by,
                reason=domain_event.reason,
            )

        elif isinstance(domain_event, domain_events.ScenarioCreated):
            return scenario_created_pb2.ScenarioCreated(
                scenario_id=domain_event.scenario_id,
                companion_id=domain_event.companion_id,
                title=domain_event.title,
                price=domain_event.price,
                duration_minutes=domain_event.duration_minutes,
            )

        elif isinstance(domain_event, domain_events.ScenarioUpdated):
            return scenario_updated_pb2.ScenarioUpdated(
                scenario_id=domain_event.scenario_id,
                companion_id=domain_event.companion_id,
                title=domain_event.title,
                price=domain_event.price,
                duration_minutes=domain_event.duration_minutes,
                status=domain_event.status,
            )

        elif isinstance(domain_event, domain_events.ScenarioDeleted):
            return scenario_deleted_pb2.ScenarioDeleted(
                scenario_id=domain_event.scenario_id,
                companion_id=domain_event.companion_id,
            )

        elif isinstance(domain_event, domain_events.VoiceIntroRegistered):
            return voice_intro_registered_pb2.VoiceIntroRegistered(
                companion_id=domain_event.companion_id,
                asset_id=domain_event.asset_id,
                file_url=domain_event.file_url,
                duration_seconds=domain_event.duration_seconds,
                size_bytes=domain_event.size_bytes,
            )

        elif isinstance(domain_event, domain_events.VoiceIntroRejected):
            return voice_intro_rejected_pb2.VoiceIntroRejected(
                companion_id=domain_event.companion_id,
                file_url=domain_event.file_url,
                reason=domain_event.reason,
            )

        elif isinstance(domain_event, domain_events.AlbumImageRegistered):
            return album_image_registered_pb2.AlbumImageRegistered(
                companion_id=domain_event.companion_id,
                asset_id=domain_event.asset_id,
                file_url=domain_event.file_url,
                size_bytes=domain_event.size_bytes,
            )

        raise ValueError(f"Unknown domain event type: {type(domain_event)}")
