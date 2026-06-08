import uuid
from typing import Dict
from internal.domain.vo import Money
from internal.domain.aggregate.wallet import Wallet
from internal.domain.aggregate.escrow import Escrow
from internal.domain.aggregate.transaction import Transaction
from internal.domain.service import CurrencyExchangeService
from internal.domain.errors import (
    WalletNotFoundError,
    EscrowAlreadyExistsError,
    EscrowNotFoundError,
)
from internal.domain.repository import (
    IWalletRepository,
    IEscrowRepository,
    ITransactionRepository,
)
from internal.application.port import IEventPublisher
from internal.infrastructure.payment.vnpay import VNPayAdapter


class FinanceCommandService:
    def __init__(
        self,
        wallet_repo: IWalletRepository,
        escrow_repo: IEscrowRepository,
        transaction_repo: ITransactionRepository,
        event_publisher: IEventPublisher,
        vnpay_adapter: VNPayAdapter,
    ):
        self.wallet_repo = wallet_repo
        self.escrow_repo = escrow_repo
        self.transaction_repo = transaction_repo
        self.event_publisher = event_publisher
        self.vnpay_adapter = vnpay_adapter

    async def get_or_create_wallet(self, user_id: str, lock: bool = False) -> Wallet:
        """
        Lazy initialization fallback logic.
        """
        wallet = await self.wallet_repo.find_by_user_id(user_id, lock=lock)
        if not wallet:
            wallet_id = str(uuid.uuid4())
            wallet = Wallet.create(wallet_id, user_id)
            await self.wallet_repo.save(wallet)
        return wallet

    async def create_wallet_onboard(self, user_id: str) -> None:
        """
        Active initialization logic when UserRegistered Kafka event is consumed.
        """
        existing = await self.wallet_repo.find_by_user_id(user_id)
        if existing:
            return  # Idempotent skip

        wallet_id = str(uuid.uuid4())
        wallet = Wallet.create(wallet_id, user_id)
        await self.wallet_repo.save(wallet)

    async def freeze_coin(self, user_id: str, amount: int, booking_id: str) -> str:
        """
        Freezes coins in Client's wallet for booking request reservation.
        """
        wallet = await self.get_or_create_wallet(user_id, lock=True)
        money = Money(amount)

        # Freeze coin and append domain event
        wallet.freeze_coin(money, booking_id)

        # Create Ledger transaction logs (PENDING)
        txn_id = str(uuid.uuid4())
        txn = Transaction.create(
            transaction_id=txn_id,
            user_id=user_id,
            amount=money,
            type="BOOKING_RESERVATION",
            status="PENDING",
            reference_id=booking_id,
        )

        await self.wallet_repo.save(wallet)
        await self.transaction_repo.save(txn)

        # Commit outbox events
        for event in wallet.clear_events():
            self.event_publisher.publish(event)

        return txn_id

    async def transfer_to_escrow(
        self, user_id: str, amount: int, booking_id: str
    ) -> str:
        """
        Transfers frozen coins from Client's wallet to Escrow fund when Companion accepts booking.
        """
        wallet = await self.wallet_repo.find_by_user_id(user_id, lock=True)
        if not wallet:
            raise WalletNotFoundError(user_id)

        money = Money(amount)

        # Ensure no active Escrow exists for this booking (INV-F04)
        existing_escrow = await self.escrow_repo.find_by_booking_id(booking_id)
        if existing_escrow and existing_escrow.status == "HELD":
            raise EscrowAlreadyExistsError(booking_id)

        # Deduct from frozen balance
        wallet.deduct_frozen(money)

        # Create Escrow fund (starts HELD)
        escrow_id = str(uuid.uuid4())
        escrow = Escrow.create(escrow_id, booking_id, money)

        # Update or record escrow deposit transaction
        txn = await self.transaction_repo.find_by_reference_id(
            booking_id, "BOOKING_RESERVATION"
        )
        if not txn:
            txn_id = str(uuid.uuid4())
            txn = Transaction.create(
                transaction_id=txn_id,
                user_id=user_id,
                amount=money,
                type="BOOKING_RESERVATION",
                status="SUCCESS",
                reference_id=booking_id,
            )
        else:
            txn.success()

        await self.wallet_repo.save(wallet)
        await self.escrow_repo.save(escrow)
        await self.transaction_repo.save(txn)

        # Commit outbox events
        for event in escrow.clear_events():
            self.event_publisher.publish(event)

        return escrow_id

    async def process_payout(
        self, booking_id: str, companion_id: str, commission_rate: float
    ) -> str:
        """
        Releases Escrow fund, deducts system commission, and pays net amount to Companion.
        """
        escrow = await self.escrow_repo.find_by_booking_id(booking_id, lock=True)
        if not escrow:
            raise EscrowNotFoundError(booking_id)

        # Process payout logic (validates HELD internally [INV-F05])
        commission, net_amount = escrow.payout(companion_id, commission_rate)

        # Credit Companion's wallet
        companion_wallet = await self.get_or_create_wallet(companion_id, lock=True)
        companion_wallet.deposit(Money(net_amount))

        # Log release transaction
        txn_id = str(uuid.uuid4())
        txn = Transaction.create(
            transaction_id=txn_id,
            user_id=companion_id,
            amount=Money(net_amount),
            type="ESCROW_RELEASE",
            status="SUCCESS",
            reference_id=booking_id,
        )

        await self.escrow_repo.save(escrow)
        await self.wallet_repo.save(companion_wallet)
        await self.transaction_repo.save(txn)

        # Commit outbox events
        for event in escrow.clear_events():
            self.event_publisher.publish(event)

        return txn_id

    async def refund_escrow(
        self, booking_id: str, client_id: str, refund_amount: int
    ) -> str:
        """
        Refunds the Escrow fund back to Client's wallet.
        """
        escrow = await self.escrow_repo.find_by_booking_id(booking_id, lock=True)
        if not escrow:
            raise EscrowNotFoundError(booking_id)

        money_refund = Money(refund_amount)

        # Refund logic (validates HELD internally [INV-F05])
        escrow.refund(client_id, money_refund)

        # Refund Client's wallet
        client_wallet = await self.get_or_create_wallet(client_id, lock=True)
        client_wallet.deposit(money_refund)

        # Log refund transaction
        txn_id = str(uuid.uuid4())
        txn = Transaction.create(
            transaction_id=txn_id,
            user_id=client_id,
            amount=money_refund,
            type="REFUND",
            status="SUCCESS",
            reference_id=booking_id,
        )

        await self.escrow_repo.save(escrow)
        await self.wallet_repo.save(client_wallet)
        await self.transaction_repo.save(txn)

        # Commit outbox events
        for event in escrow.clear_events():
            self.event_publisher.publish(event)

        return txn_id

    async def initiate_topup(
        self, user_id: str, amount_coins: int, client_ip: str
    ) -> str:
        """
        Initiates a topup request by creating a PENDING transaction and generating signed VNPay URL.
        """
        money = Money(amount_coins)
        vnd_amount = CurrencyExchangeService.RATE_COIN_TO_VND * amount_coins

        txn_id = str(uuid.uuid4())
        txn = Transaction.create(
            transaction_id=txn_id,
            user_id=user_id,
            amount=money,
            type="TOPUP",
            status="PENDING",
            reference_id=txn_id,
        )

        await self.transaction_repo.save(txn)

        # Generate signed gateway payment URL
        vnpay_url = self.vnpay_adapter.generate_payment_url(
            txn_ref=txn_id, amount_vnd=vnd_amount, ip_address=client_ip
        )
        return vnpay_url

    async def process_vnpay_ipn(self, params: Dict[str, str]) -> Dict[str, str]:
        """
        Processes VNPay IPN webhook securely, credits client's wallet on success, and logs audit ledger.
        Ensures strict transaction Idempotency.
        """
        # 1. Validate signature
        is_valid_sig = self.vnpay_adapter.validate_ipn_signature(params)
        if not is_valid_sig:
            return {"RspCode": "97", "Message": "Invalid Signature"}

        # 2. Extract transaction reference
        txn_id = params.get("vnp_TxnRef")
        if not txn_id:
            return {"RspCode": "01", "Message": "Order not found"}

        # 3. Retrieve local transaction
        txn = await self.transaction_repo.find_by_id(txn_id, lock=True)
        if not txn:
            return {"RspCode": "01", "Message": "Order not found"}

        # 4. Check Idempotency: Order must not be confirmed already
        if txn.status != "PENDING":
            return {"RspCode": "02", "Message": "Order already confirmed"}

        # 5. Verify Amount (VNPay standard param check)
        vnp_amount_cent = int(params.get("vnp_Amount", 0))
        expected_vnd_cent = (
            CurrencyExchangeService.RATE_COIN_TO_VND * txn.amount.amount * 100
        )
        if vnp_amount_cent != expected_vnd_cent:
            return {"RspCode": "04", "Message": "Invalid Amount"}

        # 6. Check transaction status returned by VNPay gateway
        vnp_response_code = params.get("vnp_ResponseCode")

        if vnp_response_code == "00":
            # SUCCESS payment
            txn.success()
            wallet = await self.get_or_create_wallet(txn.user_id, lock=True)

            # Credit wallet
            wallet.topup(
                amount=txn.amount,
                transaction_id=txn_id,
                vnpay_amount_vnd=vnp_amount_cent // 100,
            )

            await self.wallet_repo.save(wallet)
            await self.transaction_repo.save(txn)

            # Publish event
            for event in wallet.clear_events():
                self.event_publisher.publish(event)
        else:
            # FAILED payment
            txn.fail()
            await self.transaction_repo.save(txn)

        return {"RspCode": "00", "Message": "Confirm success"}
