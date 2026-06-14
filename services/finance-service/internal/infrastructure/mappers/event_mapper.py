from internal.domain import events as domain_events
from google.protobuf.timestamp_pb2 import Timestamp

# Protobuf imports will be available after generation in python path
try:
    from gen.finance.v1.events import (
        coins_frozen_pb2,
        coins_unfrozen_pb2,
        escrow_created_pb2,
        payout_processed_pb2,
        escrow_refunded_pb2,
        wallet_topped_up_pb2,
        escrow_failed_pb2,
        refund_failed_pb2,
        coins_freeze_failed_pb2,
    )
except ImportError:
    # Fallback/dynamic import wrapper if proto is not compiled yet
    coins_frozen_pb2 = None
    coins_unfrozen_pb2 = None
    escrow_created_pb2 = None
    payout_processed_pb2 = None
    escrow_refunded_pb2 = None
    wallet_topped_up_pb2 = None
    escrow_failed_pb2 = None
    refund_failed_pb2 = None
    coins_freeze_failed_pb2 = None


class EventMapper:
    @staticmethod
    def to_protobuf(domain_event: domain_events.DomainEvent):
        """
        Maps a pure domain event dataclass to its corresponding Protobuf message.
        """
        # Ensure imports are populated
        from gen.finance.v1.events import (
            coins_frozen_pb2,
            coins_unfrozen_pb2,
            escrow_created_pb2,
            payout_processed_pb2,
            escrow_refunded_pb2,
            wallet_topped_up_pb2,
            escrow_failed_pb2,
            refund_failed_pb2,
            coins_freeze_failed_pb2,
        )

        occurred_timestamp = Timestamp()
        if hasattr(domain_event, "occurred_at") and domain_event.occurred_at:
            occurred_timestamp.FromDatetime(domain_event.occurred_at)
        else:
            from datetime import datetime, timezone

            occurred_timestamp.FromDatetime(datetime.now(timezone.utc))

        import uuid

        event_id = (
            getattr(domain_event, "event_id", None)
            or getattr(domain_event, "transaction_id", None)
            or str(uuid.uuid4())
        )

        if isinstance(domain_event, domain_events.CoinsFrozen):
            return coins_frozen_pb2.CoinsFrozen(
                transaction_id=event_id,
                booking_id=domain_event.booking_id,
                user_id=domain_event.user_id,
                amount=domain_event.amount,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.CoinsUnfrozen):
            return coins_unfrozen_pb2.CoinsUnfrozen(
                transaction_id=event_id,
                booking_id=domain_event.booking_id,
                user_id=domain_event.user_id,
                amount=domain_event.amount,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.EscrowCreated):
            return escrow_created_pb2.EscrowCreated(
                booking_id=domain_event.booking_id,
                amount=domain_event.amount,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.PayoutProcessed):
            return payout_processed_pb2.PayoutProcessed(
                booking_id=domain_event.booking_id,
                companion_id=domain_event.companion_id,
                amount=domain_event.amount,
                commission_amount=domain_event.commission_amount,
                net_amount=domain_event.net_amount,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.EscrowRefunded):
            return escrow_refunded_pb2.EscrowRefunded(
                booking_id=domain_event.booking_id,
                client_id=domain_event.client_id,
                refund_amount=domain_event.refund_amount,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.WalletToppedUp):
            return wallet_topped_up_pb2.WalletToppedUp(
                transaction_id=event_id,
                user_id=domain_event.user_id,
                amount=domain_event.amount,
                vnpay_amount_vnd=domain_event.vnpay_amount_vnd,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.EscrowFailed):
            return escrow_failed_pb2.EscrowFailed(
                booking_id=domain_event.booking_id,
                client_id=domain_event.client_id,
                reason=domain_event.reason,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.RefundFailed):
            return refund_failed_pb2.RefundFailed(
                booking_id=domain_event.booking_id,
                client_id=domain_event.client_id,
                reason=domain_event.reason,
                occurred_at=occurred_timestamp,
            )

        elif isinstance(domain_event, domain_events.CoinsFreezeFailed):
            return coins_freeze_failed_pb2.CoinsFreezeFailed(
                booking_id=domain_event.booking_id,
                user_id=domain_event.user_id,
                amount=domain_event.amount,
                reason=domain_event.reason,
                occurred_at=occurred_timestamp,
            )

        raise ValueError(f"Unknown domain event type: {type(domain_event)}")
