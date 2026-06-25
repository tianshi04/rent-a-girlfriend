import grpc
import logging
from typing import Dict
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker
from internal.bootstrap import bootstrap_services
from gen.profile.v1.service import profile_service_pb2_grpc
from gen.profile.v1.messages.scenario_command_response_pb2 import (
    ScenarioCommandResponse,
)
from gen.profile.v1.messages.media_command_response_pb2 import MediaCommandResponse
from gen.profile.v1.messages.scenario_snapshot_response_pb2 import (
    ScenarioSnapshotResponse,
)
from internal.domain.errors import DomainError

logger = logging.getLogger("grpc_servicer")


class ProfileServiceServicer(profile_service_pb2_grpc.ProfileServiceServicer):
    def __init__(self, session_factory: async_sessionmaker[AsyncSession]):
        self.session_factory = session_factory

    def _extract_auth_headers(self, context) -> Dict[str, str]:
        """
        Extract authenticated headers injected by Istio Waypoint.
        """
        metadata = dict(context.invocation_metadata())
        return {
            "user_id": metadata.get("user-id", ""),
            "user_role": metadata.get("user-role", ""),
            "user_status": metadata.get("user-status", ""),
            "user_email": metadata.get("user-email", ""),
        }

    def _handle_exception(self, context, e: Exception):
        logger.error(f"gRPC service error: {e}", exc_info=True)
        if isinstance(e, DomainError):
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(e))
        else:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("Internal server error")

    # --- Profile Commands ---

    # --- Scenario Commands ---

    async def CreateScenario(self, request, context):
        try:
            auth_info = self._extract_auth_headers(context)
            companion_id = request.companion_id or auth_info.get("user_id")

            async with self.session_factory() as session:
                _, scenario_cmd, _, _ = bootstrap_services(session)
                scenario_id = await scenario_cmd.create_scenario(
                    companion_id=companion_id,
                    title=request.title,
                    description=request.description,
                    price=request.price,
                    duration_minutes=request.duration_minutes,
                )
                await session.commit()

            return ScenarioCommandResponse(
                scenario_id=scenario_id,
                status="SUCCESS",
                message="Scenario created successfully",
            )
        except Exception as e:
            self._handle_exception(context, e)
            return ScenarioCommandResponse(status="FAILED")

    async def UpdateScenario(self, request, context):
        try:
            auth_info = self._extract_auth_headers(context)
            companion_id = request.companion_id or auth_info.get("user_id")

            async with self.session_factory() as session:
                _, scenario_cmd, _, _ = bootstrap_services(session)
                await scenario_cmd.update_scenario(
                    scenario_id=request.scenario_id,
                    companion_id=companion_id,
                    title=request.title,
                    description=request.description,
                    price=request.price,
                    duration_minutes=request.duration_minutes,
                    status=request.status,
                )
                await session.commit()

            return ScenarioCommandResponse(
                scenario_id=request.scenario_id,
                status="SUCCESS",
                message="Scenario updated successfully",
            )
        except Exception as e:
            self._handle_exception(context, e)
            return ScenarioCommandResponse(status="FAILED")

    async def DeleteScenario(self, request, context):
        try:
            auth_info = self._extract_auth_headers(context)
            companion_id = request.companion_id or auth_info.get("user_id")

            async with self.session_factory() as session:
                _, scenario_cmd, _, _ = bootstrap_services(session)
                await scenario_cmd.delete_scenario(
                    scenario_id=request.scenario_id, companion_id=companion_id
                )
                await session.commit()

            return ScenarioCommandResponse(
                scenario_id=request.scenario_id,
                status="SUCCESS",
                message="Scenario deleted successfully",
            )
        except Exception as e:
            self._handle_exception(context, e)
            return ScenarioCommandResponse(status="FAILED")

    # --- Media Commands ---

    async def RegisterVoiceIntro(self, request, context):
        try:
            auth_info = self._extract_auth_headers(context)
            companion_id = request.companion_id or auth_info.get("user_id")

            async with self.session_factory() as session:
                _, _, media_cmd, _ = bootstrap_services(session)
                asset_id = await media_cmd.register_voice_intro(
                    companion_id=companion_id,
                    file_url=request.file_url,
                    duration_seconds=request.duration_seconds,
                    size_bytes=request.size_bytes,
                )
                await session.commit()

            return MediaCommandResponse(
                asset_id=asset_id,
                status="APPROVED",
                message="Voice intro registered successfully",
            )
        except Exception as e:
            self._handle_exception(context, e)
            return MediaCommandResponse(status="FAILED")

    async def RegisterAlbumImage(self, request, context):
        try:
            auth_info = self._extract_auth_headers(context)
            companion_id = request.companion_id or auth_info.get("user_id")

            async with self.session_factory() as session:
                _, _, media_cmd, _ = bootstrap_services(session)
                asset_id = await media_cmd.register_album_image(
                    companion_id=companion_id,
                    file_url=request.file_url,
                    size_bytes=request.size_bytes,
                )
                await session.commit()

            return MediaCommandResponse(
                asset_id=asset_id,
                status="APPROVED",
                message="Album image registered successfully",
            )
        except Exception as e:
            self._handle_exception(context, e)
            return MediaCommandResponse(status="FAILED")

    # --- Internal Queries ---

    async def GetScenarioSnapshot(self, request, context):
        try:
            async with self.session_factory() as session:
                _, _, _, query_service = bootstrap_services(session)
                snapshot = await query_service.get_scenario_snapshot(
                    request.scenario_id
                )

            return ScenarioSnapshotResponse(
                scenario_id=snapshot["scenario_id"],
                companion_id=snapshot["companion_id"],
                title=snapshot["title"],
                price=snapshot["price"],
                duration_minutes=snapshot["duration_minutes"],
            )
        except Exception as e:
            self._handle_exception(context, e)
            return ScenarioSnapshotResponse()
