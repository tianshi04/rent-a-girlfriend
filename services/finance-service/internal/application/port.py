from abc import ABC, abstractmethod
from internal.domain.events import DomainEvent


class IEventPublisher(ABC):
    @abstractmethod
    def publish(self, event: DomainEvent) -> None:
        pass
