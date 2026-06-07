import grpc
import logging
from typing import Dict
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker
from internal.domain.errors import DomainError
from gen.dispute.v1.service import dispute_service_pb2_grpc
from gen.dispute.v1.messages.dispute_command_response_pb2 import DisputeCommandResponse

logger = logging.getLogger("grpc_servicer")


class DisputeServiceServicer(dispute_service_pb2_grpc.DisputeServiceServicer):
    def __init__(
        self, session_factory: async_sessionmaker[AsyncSession], container: any = None
    ):
        self.session_factory = session_factory
        # container will be used to resolve the command service if bootstrap is already configured
        self.container = container

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

    async def CreateReport(self, request, context):
        try:
            auth_info = self._extract_auth_headers(context)
            reporter_id = request.reporter_id or auth_info.get("user_id")

            if not reporter_id:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details("Reporter identity missing")
                return DisputeCommandResponse()

            # Map proto evidences to dictionaries
            evidences = [
                {"evidence_type": ev.type, "content": ev.content}
                for ev in request.evidences
            ]

            from internal.bootstrap import bootstrap_services

            async with self.session_factory() as session:
                cmd_service, _ = bootstrap_services(session)
                dispute_id = await cmd_service.create_report(
                    booking_id=request.booking_id,
                    reporter_id=reporter_id,
                    accused_id=request.accused_id,
                    reason=request.reason,
                    evidences=evidences,
                )
                await session.commit()

            return DisputeCommandResponse(
                dispute_id=dispute_id,
                status="SUCCESS",
                message="Dispute report created successfully",
            )
        except Exception as e:
            self._handle_exception(context, e)
            return DisputeCommandResponse(status="FAILED", message=str(e))

    async def ResolveDispute(self, request, context):
        try:
            auth_info = self._extract_auth_headers(context)
            # Enforce admin permission check
            if auth_info.get("user_role") != "ADMIN":
                context.set_code(grpc.StatusCode.PERMISSION_DENIED)
                context.set_details("Admin only operation")
                return DisputeCommandResponse()

            admin_id = request.admin_id or auth_info.get("user_id")
            if not admin_id:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details("Admin identity missing")
                return DisputeCommandResponse()

            from internal.bootstrap import bootstrap_services

            async with self.session_factory() as session:
                cmd_service, _ = bootstrap_services(session)
                await cmd_service.resolve_dispute(
                    dispute_id=request.dispute_id,
                    admin_id=admin_id,
                    resolution=request.resolution,
                    notes=request.notes,
                )
                await session.commit()

            return DisputeCommandResponse(
                dispute_id=request.dispute_id,
                status="SUCCESS",
                message=f"Dispute resolved with action {request.resolution}",
            )
        except Exception as e:
            self._handle_exception(context, e)
            return DisputeCommandResponse(status="FAILED", message=str(e))
