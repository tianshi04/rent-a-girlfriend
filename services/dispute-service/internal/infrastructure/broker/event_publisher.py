import json
import uuid
from google.protobuf.json_format import MessageToDict
from sqlalchemy.ext.asyncio import AsyncSession
from internal.domain.events import DomainEvent
from internal.application.port import IEventPublisher
from internal.infrastructure.persistence.models import OutboxModel
from internal.infrastructure.mappers.event_mapper import EventMapper


class DatabaseEventPublisher(IEventPublisher):
    """
    Saves events directly to the Outbox table in the database.
    Part of the atomic business transaction.
    """

    def __init__(self, session: AsyncSession):
        self.session = session

    def publish(self, event: DomainEvent) -> None:
        # Convert pure domain event to generated Protobuf message
        try:
            proto_msg = EventMapper.to_protobuf(event)
        except ValueError:
            # SAGA internal step events (SagaStepCompleted, etc.) do not have proto contracts
            # and do not need to be published to Kafka. We only publish external events.
            return

        # Strictly generate JSON payload based on proto contract
        payload_dict = MessageToDict(
            proto_msg, preserving_proto_field_name=True, use_integers_for_enums=True
        )

        event_id = str(uuid.uuid4())

        import re
        kebab_name = re.sub('(.)([A-Z][a-z]+)', r'\1-\2', proto_msg.DESCRIPTOR.name)
        kebab_name = re.sub('([a-z0-9])([A-Z])', r'\1-\2', kebab_name).lower()

        outbox = OutboxModel(
            event_id=event_id,
            event_type=f"dispute.{kebab_name}.v1",
            payload=json.dumps(payload_dict),
        )
        self.session.add(outbox)
