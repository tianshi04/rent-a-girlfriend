from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from internal.domain.aggregate.wallet import Wallet
from internal.domain.aggregate.escrow import Escrow
from internal.domain.aggregate.transaction import Transaction
from internal.domain.vo import Money
from internal.domain.repository import (
    IWalletRepository,
    IEscrowRepository,
    ITransactionRepository,
)
from internal.infrastructure.persistence.models import (
    WalletModel,
    EscrowModel,
    TransactionModel,
)


class WalletRepository(IWalletRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, wallet: Wallet) -> None:
        model = WalletModel(
            wallet_id=wallet.wallet_id,
            user_id=wallet.user_id,
            available_balance=wallet.available_balance.amount,
            frozen_balance=wallet.frozen_balance.amount,
        )
        await self.session.merge(model)
        await self.session.flush()

    async def find_by_id(self, wallet_id: str) -> Optional[Wallet]:
        stmt = select(WalletModel).filter_by(wallet_id=wallet_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    async def find_by_user_id(self, user_id: str) -> Optional[Wallet]:
        stmt = select(WalletModel).filter_by(user_id=user_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    def _to_domain(self, model: WalletModel) -> Wallet:
        return Wallet(
            wallet_id=model.wallet_id,
            user_id=model.user_id,
            available_balance=Money(model.available_balance),
            frozen_balance=Money(model.frozen_balance),
        )


class EscrowRepository(IEscrowRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, escrow: Escrow) -> None:
        model = EscrowModel(
            escrow_id=escrow.escrow_id,
            booking_id=escrow.booking_id,
            amount=escrow.amount.amount,
            status=escrow.status,
        )
        await self.session.merge(model)
        await self.session.flush()

    async def find_by_id(self, escrow_id: str) -> Optional[Escrow]:
        stmt = select(EscrowModel).filter_by(escrow_id=escrow_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    async def find_by_booking_id(self, booking_id: str) -> Optional[Escrow]:
        stmt = select(EscrowModel).filter_by(booking_id=booking_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    def _to_domain(self, model: EscrowModel) -> Escrow:
        return Escrow(
            escrow_id=model.escrow_id,
            booking_id=model.booking_id,
            amount=Money(model.amount),
            status=model.status,
        )


class TransactionRepository(ITransactionRepository):
    def __init__(self, session: AsyncSession):
        self.session = session

    async def save(self, transaction: Transaction) -> None:
        model = TransactionModel(
            transaction_id=transaction.transaction_id,
            user_id=transaction.user_id,
            amount=transaction.amount.amount,
            type=transaction.type,
            status=transaction.status,
            reference_id=transaction.reference_id,
        )
        await self.session.merge(model)
        await self.session.flush()

    async def find_by_id(self, transaction_id: str) -> Optional[Transaction]:
        stmt = select(TransactionModel).filter_by(transaction_id=transaction_id)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    async def find_by_reference_id(
        self, reference_id: str, type: str
    ) -> Optional[Transaction]:
        stmt = select(TransactionModel).filter_by(reference_id=reference_id, type=type)
        result = await self.session.execute(stmt)
        model = result.scalars().first()
        if not model:
            return None
        return self._to_domain(model)

    def _to_domain(self, model: TransactionModel) -> Transaction:
        return Transaction(
            transaction_id=model.transaction_id,
            user_id=model.user_id,
            amount=Money(model.amount),
            type=model.type,
            status=model.status,
            reference_id=model.reference_id,
        )
