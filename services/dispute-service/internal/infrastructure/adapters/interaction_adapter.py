import logging
import grpc
from internal.application.port import IInteractionPort

# Import generated gRPC code
try:
    from gen.interaction.v1.service.interaction_service_pb2_grpc import (
        InteractionServiceStub,
    )
    from gen.interaction.v1.messages.lock_chat_room_request_pb2 import (
        LockChatRoomRequest,
    )
    from gen.interaction.v1.messages.hide_review_request_pb2 import HideReviewRequest
except ImportError:
    # Fallback to avoid import failures if protos are not generated in some environments
    InteractionServiceStub = None
    LockChatRoomRequest = None
    HideReviewRequest = None

logger = logging.getLogger("interaction_adapter")


class MockInteractionAdapter(IInteractionPort):
    """
    Mock/Stub adapter for Interaction Service.
    Returns success by default.
    """

    async def hide_review_and_lock_chat(self, booking_id: str) -> bool:
        # Mocking successful hide review and lock chat call
        return True

    async def lock_chat_room(self, booking_id: str) -> bool:
        # Mocking successful lock chat call
        return True


class gRPCInteractionAdapter(IInteractionPort):
    """
    Real gRPC adapter for Interaction Service.
    """

    def __init__(self, address: str):
        self.address = address

    async def hide_review_and_lock_chat(self, booking_id: str) -> bool:
        if InteractionServiceStub is None:
            logger.error(
                "Interaction gRPC stubs not found. Make sure protos are generated."
            )
            return False

        try:
            logger.info(
                f"Calling Interaction Service to HideReview and LockChatRoom via gRPC at {self.address} for booking {booking_id}"
            )
            async with grpc.aio.insecure_channel(self.address) as channel:
                stub = InteractionServiceStub(channel)

                # 1. Hide client review
                hide_request = HideReviewRequest(
                    booking_id=booking_id, reason="Dispute resolved with client refund"
                )
                hide_response = await stub.HideReview(hide_request)
                logger.info(
                    f"HideReview status: {hide_response.status}, message: {hide_response.message}"
                )

                # 2. Lock chat room
                lock_request = LockChatRoomRequest(booking_id=booking_id)
                lock_response = await stub.LockChatRoom(lock_request)
                logger.info(
                    f"LockChatRoom status: {lock_response.status}, message: {lock_response.message}"
                )

                return (
                    hide_response.status != "FAILED"
                    and lock_response.status != "FAILED"
                )
        except Exception as e:
            logger.error(f"gRPC calls to Interaction Service failed: {e}")
            return False

    async def lock_chat_room(self, booking_id: str) -> bool:
        if InteractionServiceStub is None:
            logger.error(
                "Interaction gRPC stubs not found. Make sure protos are generated."
            )
            return False

        try:
            logger.info(
                f"Calling Interaction Service to LockChatRoom via gRPC at {self.address} for booking {booking_id}"
            )
            async with grpc.aio.insecure_channel(self.address) as channel:
                stub = InteractionServiceStub(channel)
                request = LockChatRoomRequest(booking_id=booking_id)
                response = await stub.LockChatRoom(request)
                logger.info(
                    f"LockChatRoom status: {response.status}, message: {response.message}"
                )
                return response.status != "FAILED"
        except Exception as e:
            logger.error(f"gRPC call to Interaction Service LockChatRoom failed: {e}")
            return False
