import uuid
import logging
import grpc
from internal.application.port import IFinancePort, RefundResult, PayoutResult

# Import generated gRPC code
try:
    from gen.finance.v1.service.finance_service_pb2_grpc import FinanceServiceStub
    from gen.finance.v1.messages.refund_escrow_request_pb2 import RefundEscrowRequest
    from gen.finance.v1.messages.process_payout_request_pb2 import ProcessPayoutRequest
except ImportError:
    # Fallback to avoid import failures if protos are not generated in some environments
    FinanceServiceStub = None
    RefundEscrowRequest = None
    ProcessPayoutRequest = None

logger = logging.getLogger("finance_adapter")


class MockFinanceAdapter(IFinancePort):
    """
    Mock/Stub adapter for Finance Service.
    Returns success by default.
    """

    async def refund_escrow_to_wallet(self, booking_id: str) -> RefundResult:
        # Mocking successful refund call
        return RefundResult(success=True, transaction_id=str(uuid.uuid4()))

    async def payout_from_escrow(
        self, booking_id: str, companion_wallet_id: str, commission_rate: float
    ) -> PayoutResult:
        # Mocking successful payout call
        return PayoutResult(success=True, transaction_id=str(uuid.uuid4()))

    async def get_payout_snapshot(self, booking_id: str) -> tuple[str, float]:
        # Mock snapshot fetch
        return (f"wallet-{booking_id}", 0.15)


class gRPCFinanceAdapter(IFinancePort):
    """
    Real gRPC adapter for Finance Service.
    """

    def __init__(self, address: str):
        self.address = address

    async def refund_escrow_to_wallet(self, booking_id: str) -> RefundResult:
        if FinanceServiceStub is None:
            logger.error(
                "Finance gRPC stubs not found. Make sure protos are generated."
            )
            return RefundResult(success=False, error="gRPC stubs not found")

        try:
            logger.info(
                f"Calling Finance Service RefundEscrow via gRPC at {self.address} for booking {booking_id}"
            )
            async with grpc.aio.insecure_channel(self.address) as channel:
                stub = FinanceServiceStub(channel)
                # Note: RefundEscrowRequest in protobuf has:
                # string booking_id = 1;
                # string client_id = 2;
                # int64 refund_amount = 3;
                # In Dispute SAGA, booking_id is the primary key and the finance service escrow
                # handles resolving the matching client and amount. We pass client_id="" and refund_amount=0.
                request = RefundEscrowRequest(
                    booking_id=booking_id, client_id="", refund_amount=0
                )
                response = await stub.RefundEscrow(request)
                if response.status == "FAILED":
                    return RefundResult(success=False, error=response.message)
                return RefundResult(
                    success=True, transaction_id=response.transaction_id
                )
        except Exception as e:
            logger.error(f"gRPC call to Finance Service RefundEscrow failed: {e}")
            return RefundResult(success=False, error=str(e))

    async def payout_from_escrow(
        self, booking_id: str, companion_wallet_id: str, commission_rate: float
    ) -> PayoutResult:
        if FinanceServiceStub is None:
            logger.error(
                "Finance gRPC stubs not found. Make sure protos are generated."
            )
            return PayoutResult(success=False, error="gRPC stubs not found")

        try:
            logger.info(
                f"Calling Finance Service ProcessPayout via gRPC at {self.address} for booking {booking_id}"
            )
            async with grpc.aio.insecure_channel(self.address) as channel:
                stub = FinanceServiceStub(channel)
                request = ProcessPayoutRequest(
                    booking_id=booking_id,
                    companion_id=companion_wallet_id,
                    commission_rate=commission_rate,
                )
                response = await stub.ProcessPayout(request)
                if response.status == "FAILED":
                    return PayoutResult(success=False, error=response.message)
                return PayoutResult(
                    success=True, transaction_id=response.transaction_id
                )
        except Exception as e:
            logger.error(f"gRPC call to Finance Service ProcessPayout failed: {e}")
            return PayoutResult(success=False, error=str(e))

    async def get_payout_snapshot(self, booking_id: str) -> tuple[str, float]:
        # Stub implementation for get_payout_snapshot via gRPC
        # In a real scenario, this would call a Finance or Booking service method.
        # For now, returning stub values as we lack the specific gRPC definition.
        logger.info(f"Stubbing get_payout_snapshot for booking {booking_id}")
        return (f"wallet-{booking_id}", 0.15)
