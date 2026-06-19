from abc import ABC, abstractmethod
from typing import Optional
from internal.domain.aggregate.wallet import Wallet
from internal.domain.aggregate.escrow import Escrow
from internal.domain.aggregate.transaction import Transaction
from internal.domain.vo import TransactionType


class IWalletRepository(ABC):
    @abstractmethod
    async def save(self, wallet: Wallet) -> None:
        pass

    @abstractmethod
    async def find_by_id(self, wallet_id: str, lock: bool = False) -> Optional[Wallet]:
        pass

    @abstractmethod
    async def find_by_user_id(
        self, user_id: str, lock: bool = False
    ) -> Optional[Wallet]:
        pass


class IEscrowRepository(ABC):
    @abstractmethod
    async def save(self, escrow: Escrow) -> None:
        pass

    @abstractmethod
    async def find_by_id(self, escrow_id: str, lock: bool = False) -> Optional[Escrow]:
        pass

    @abstractmethod
    async def find_by_booking_id(
        self, booking_id: str, lock: bool = False
    ) -> Optional[Escrow]:
        pass


class ITransactionRepository(ABC):
    @abstractmethod
    async def save(self, transaction: Transaction) -> None:
        pass

    @abstractmethod
    async def find_by_id(
        self, transaction_id: str, lock: bool = False
    ) -> Optional[Transaction]:
        pass

    @abstractmethod
    async def find_by_reference_id(
        self, reference_id: str, type: TransactionType, lock: bool = False
    ) -> Optional[Transaction]:
        pass
