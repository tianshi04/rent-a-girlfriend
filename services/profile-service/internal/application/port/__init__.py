from abc import ABC, abstractmethod
from typing import Dict, Any
from internal.domain.events import DomainEvent


class IStoragePort(ABC):
    @abstractmethod
    def generate_presigned_put_url(
        self, key: str, content_type: str, content_length: int
    ) -> str:
        """
        Generate S3 presigned PUT URL with content-length-range policy.
        Completely S3 standard to avoid vendor lock-in.
        """
        pass

    @abstractmethod
    def get_object_metadata(self, key: str) -> Dict[str, Any]:
        """
        Retrieve object size and metadata from S3 using HeadObject API.
        Used for the double-check verification step.
        """
        pass


class IEventPublisher(ABC):
    @abstractmethod
    def publish(self, event: DomainEvent) -> None:
        """
        Publish event to broker. Under hexagonal, this is handled
        via Transactional Outbox pattern.
        """
        pass
