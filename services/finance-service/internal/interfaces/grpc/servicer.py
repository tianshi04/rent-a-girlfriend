import grpc
import logging
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from finance.v1.service import finance_service_pb2_grpc
from finance.v1.messages import (
    finance_command_response_pb2,
    get_wallet_response_pb2,
    check_balance_response_pb2,
)

from internal.bootstrap import bootstrap_services
from internal.domain.errors import (
    WalletNotFoundError,
    EscrowNotFoundError,
    InsufficientBalanceError,
    EscrowAlreadyExistsError,
    InvalidEscrowStatusTransitionError,
    InvalidAmountError,
)

logger = logging.getLogger("grpc_servicer")


class FinanceServiceServicer(finance_service_pb2_grpc.FinanceServiceServicer):
    def __init__(self, session_factory: async_sessionmaker[AsyncSession]):
        self.session_factory = session_factory

    async def FreezeCoin(self, request, context):
        logger.info(
            f"gRPC FreezeCoin: user_id={request.user_id}, amount={request.amount}, booking_id={request.booking_id}"
        )
        async with self.session_factory() as session:
            try:
                cmd_service = bootstrap_services(session)
                txn_id = await cmd_service.freeze_coin(
                    user_id=request.user_id,
                    amount=request.amount,
                    booking_id=request.booking_id,
                )
                await session.commit()
                return finance_command_response_pb2.FinanceCommandResponse(
                    transaction_id=txn_id,
                    status="SUCCESS",
                    message="Coins successfully frozen for booking reservation.",
                )
            except Exception as e:
                if isinstance(e, WalletNotFoundError):
                    context.set_code(grpc.StatusCode.NOT_FOUND)
                elif isinstance(e, (InsufficientBalanceError, InvalidAmountError)):
                    context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                else:
                    logger.error(f"Error freezing coins: {e}", exc_info=True)
                    context.set_code(grpc.StatusCode.INTERNAL)
                    context.set_details(
                        "Internal server error occurred." if not str(e) else str(e)
                    )
                    return finance_command_response_pb2.FinanceCommandResponse()

                context.set_details(str(e))
                return finance_command_response_pb2.FinanceCommandResponse()

    async def TransferToEscrow(self, request, context):
        logger.info(
            f"gRPC TransferToEscrow: user_id={request.user_id}, booking_id={request.booking_id}, amount={request.amount}"
        )
        async with self.session_factory() as session:
            try:
                cmd_service = bootstrap_services(session)
                escrow_id = await cmd_service.transfer_to_escrow(
                    user_id=request.user_id,
                    amount=request.amount,
                    booking_id=request.booking_id,
                )
                await session.commit()
                return finance_command_response_pb2.FinanceCommandResponse(
                    transaction_id=escrow_id,
                    status="SUCCESS",
                    message="Coins successfully transferred to Escrow HELD status.",
                )
            except WalletNotFoundError as e:
                await session.rollback()
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(str(e))
                return finance_command_response_pb2.FinanceCommandResponse()
            except EscrowAlreadyExistsError as e:
                await session.rollback()
                context.set_code(grpc.StatusCode.ALREADY_EXISTS)
                context.set_details(str(e))
                return finance_command_response_pb2.FinanceCommandResponse()
            except (InsufficientBalanceError, InvalidAmountError) as e:
                await session.rollback()
                context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                context.set_details(str(e))
                return finance_command_response_pb2.FinanceCommandResponse()
            except Exception as e:
                await session.rollback()
                logger.error(f"Error transferring to escrow: {e}", exc_info=True)
                context.set_code(grpc.StatusCode.INTERNAL)
                context.set_details("Internal server error occurred.")
                return finance_command_response_pb2.FinanceCommandResponse()

    async def ProcessPayout(self, request, context):
        logger.info(
            f"gRPC ProcessPayout: booking_id={request.booking_id}, companion_id={request.companion_id}, commission_rate={request.commission_rate}"
        )
        async with self.session_factory() as session:
            try:
                cmd_service = bootstrap_services(session)
                txn_id = await cmd_service.process_payout(
                    booking_id=request.booking_id,
                    companion_id=request.companion_id,
                    commission_rate=request.commission_rate,
                )
                await session.commit()
                return finance_command_response_pb2.FinanceCommandResponse(
                    transaction_id=txn_id,
                    status="SUCCESS",
                    message="Escrow released and payout successfully processed.",
                )
            except EscrowNotFoundError as e:
                await session.rollback()
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(str(e))
                return finance_command_response_pb2.FinanceCommandResponse()
            except InvalidEscrowStatusTransitionError as e:
                await session.rollback()
                context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                context.set_details(str(e))
                return finance_command_response_pb2.FinanceCommandResponse()
            except Exception as e:
                await session.rollback()
                logger.error(f"Error processing payout: {e}", exc_info=True)
                context.set_code(grpc.StatusCode.INTERNAL)
                context.set_details("Internal server error occurred.")
                return finance_command_response_pb2.FinanceCommandResponse()

    async def RefundEscrow(self, request, context):
        logger.info(
            f"gRPC RefundEscrow: booking_id={request.booking_id}, client_id={request.client_id}, refund_amount={request.refund_amount}"
        )
        async with self.session_factory() as session:
            try:
                cmd_service = bootstrap_services(session)
                txn_id = await cmd_service.refund_escrow(
                    booking_id=request.booking_id,
                    client_id=request.client_id,
                    refund_amount=request.refund_amount,
                )
                await session.commit()
                return finance_command_response_pb2.FinanceCommandResponse(
                    transaction_id=txn_id,
                    status="SUCCESS",
                    message="Escrow successfully refunded to client wallet.",
                )
            except Exception as e:
                if isinstance(e, EscrowNotFoundError):
                    context.set_code(grpc.StatusCode.NOT_FOUND)
                elif isinstance(
                    e,
                    (
                        InvalidEscrowStatusTransitionError,
                        InsufficientBalanceError,
                        InvalidAmountError,
                    ),
                ):
                    context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                else:
                    logger.error(f"Error refunding escrow: {e}", exc_info=True)
                    context.set_code(grpc.StatusCode.INTERNAL)
                    context.set_details(
                        "Internal server error occurred." if not str(e) else str(e)
                    )
                    return finance_command_response_pb2.FinanceCommandResponse()

                context.set_details(str(e))
                return finance_command_response_pb2.FinanceCommandResponse()

    async def GetWallet(self, request, context):
        logger.info(f"gRPC GetWallet: user_id={request.user_id}")
        async with self.session_factory() as session:
            try:
                cmd_service = bootstrap_services(session)
                # Lazy initialization logic is encapsulated here to ensure robustness
                wallet = await cmd_service.get_or_create_wallet(request.user_id)
                await session.commit()
                return get_wallet_response_pb2.GetWalletResponse(
                    wallet_id=wallet.wallet_id,
                    user_id=wallet.user_id,
                    available_balance=wallet.available_balance.amount,
                    frozen_balance=wallet.frozen_balance.amount,
                )
            except Exception as e:
                await session.rollback()
                logger.error(f"Error fetching/creating wallet: {e}", exc_info=True)
                context.set_code(grpc.StatusCode.INTERNAL)
                context.set_details("Internal server error occurred.")
                return get_wallet_response_pb2.GetWalletResponse()

    async def CheckBalance(self, request, context):
        logger.info(
            f"gRPC CheckBalance: user_id={request.user_id}, amount={request.amount}"
        )
        async with self.session_factory() as session:
            try:
                cmd_service = bootstrap_services(session)
                has_sufficient = await cmd_service.check_balance(
                    user_id=request.user_id,
                    amount=request.amount,
                )
                await session.commit()
                return check_balance_response_pb2.CheckBalanceResponse(
                    has_sufficient_balance=has_sufficient
                )
            except Exception as e:
                await session.rollback()
                if isinstance(e, InvalidAmountError):
                    context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                    context.set_details(str(e))
                else:
                    logger.error(f"Error checking balance: {e}", exc_info=True)
                    context.set_code(grpc.StatusCode.INTERNAL)
                    context.set_details("Internal server error occurred.")
                return check_balance_response_pb2.CheckBalanceResponse()
