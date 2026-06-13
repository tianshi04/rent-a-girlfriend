from datetime import datetime, timezone
from google.protobuf.timestamp_pb2 import Timestamp
from internal.domain import events as domain_events
from gen.dispute.v1.events import (
    dispute_created_pb2,
    dispute_resolved_pb2,
)


class EventMapper:
    @staticmethod
    def to_protobuf(domain_event: domain_events.DomainEvent):
        """
        Maps a domain event (dataclass) to its corresponding protobuf message.
        """
        timestamp = Timestamp()
        timestamp.FromDatetime(datetime.now(timezone.utc))

        if isinstance(domain_event, domain_events.ReportCreated):
            return dispute_created_pb2.DisputeCreated(
                dispute_id=domain_event.dispute_id,
                booking_id=domain_event.booking_id,
                reporter_id=domain_event.reporter_id,
                reason=domain_event.reason,
                occurred_at=timestamp,
            )

        elif isinstance(domain_event, domain_events.DisputeResolvedRefund):
            return dispute_resolved_pb2.DisputeResolved(
                dispute_id=domain_event.dispute_id,
                booking_id=domain_event.booking_id,
                resolution="REFUND_CLIENT",
                occurred_at=timestamp,
                resolved_by=domain_event.admin_id,
                reporter_id=domain_event.reporter_id,
                accused_id=domain_event.accused_id,
            )

        elif isinstance(domain_event, domain_events.DisputeResolvedPayout):
            return dispute_resolved_pb2.DisputeResolved(
                dispute_id=domain_event.dispute_id,
                booking_id=domain_event.booking_id,
                resolution="PAYOUT_COMPANION",
                occurred_at=timestamp,
                resolved_by=domain_event.admin_id,
                reporter_id=domain_event.reporter_id,
                accused_id=domain_event.accused_id,
            )

        elif isinstance(domain_event, domain_events.DisputeRejected):
            return dispute_resolved_pb2.DisputeResolved(
                dispute_id=domain_event.dispute_id,
                booking_id=domain_event.booking_id,
                resolution="REJECT",
                occurred_at=timestamp,
                resolved_by=domain_event.admin_id,
                reporter_id=domain_event.reporter_id,
                accused_id=domain_event.accused_id,
            )

        raise ValueError(f"Unknown domain event type: {type(domain_event)}")
