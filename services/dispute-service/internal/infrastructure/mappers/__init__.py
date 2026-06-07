from typing import Union
from internal.domain.aggregate import (
    Dispute,
    DisputeEvidence,
    DisputeRefundSaga,
    DisputePayoutSaga,
)
from internal.domain.vo import DisputeReason
from internal.infrastructure.persistence.models import (
    DisputeModel,
    DisputeEvidenceModel,
    SagaStateModel,
)


class DisputeMapper:
    @staticmethod
    def to_model(domain: Dispute) -> DisputeModel:
        model = DisputeModel(
            dispute_id=domain.dispute_id,
            booking_id=domain.booking_id,
            reporter_id=domain.reporter_id,
            accused_id=domain.accused_id,
            reason=str(domain.reason),
            status=domain.status,
            admin_id=domain.admin_id,
            resolution=domain.resolution,
            notes=domain.notes,
            version=domain.version,
        )
        model.evidences = [
            DisputeEvidenceModel(
                evidence_id=ev.evidence_id,
                dispute_id=domain.dispute_id,
                evidence_type=ev.evidence_type,
                content=ev.content,
            )
            for ev in domain.evidences
        ]
        return model

    @staticmethod
    def to_domain(model: DisputeModel) -> Dispute:
        evidences = [
            DisputeEvidence(
                evidence_id=ev.evidence_id,
                evidence_type=ev.evidence_type,
                content=ev.content,
            )
            for ev in model.evidences
        ]
        return Dispute(
            dispute_id=model.dispute_id,
            booking_id=model.booking_id,
            reporter_id=model.reporter_id,
            accused_id=model.accused_id,
            reason=DisputeReason(model.reason),
            status=model.status,
            admin_id=model.admin_id,
            resolution=model.resolution,
            notes=model.notes,
            evidences=evidences,
            version=model.version,
        )


class SagaStateMapper:
    @staticmethod
    def to_model(domain: Union[DisputeRefundSaga, DisputePayoutSaga]) -> SagaStateModel:
        saga_type = "REFUND" if isinstance(domain, DisputeRefundSaga) else "PAYOUT"

        companion_wallet_id = None
        commission_rate = None
        if saga_type == "PAYOUT":
            companion_wallet_id = getattr(domain, "companion_wallet_id", None)
            commission_rate = getattr(domain, "commission_rate", None)

        return SagaStateModel(
            saga_id=domain.saga_id,
            dispute_id=domain.dispute_id,
            booking_id=domain.booking_id,
            saga_type=saga_type,
            current_state=domain.current_state,
            retry_count=domain.retry_count,
            last_error=domain.last_error,
            version=domain.version,
            companion_wallet_id=companion_wallet_id,
            commission_rate=commission_rate,
        )

    @staticmethod
    def to_domain(model: SagaStateModel) -> Union[DisputeRefundSaga, DisputePayoutSaga]:
        if model.saga_type == "REFUND":
            return DisputeRefundSaga(
                saga_id=model.saga_id,
                dispute_id=model.dispute_id,
                booking_id=model.booking_id,
                current_state=model.current_state,
                retry_count=model.retry_count,
                last_error=model.last_error,
                version=model.version,
            )
        elif model.saga_type == "PAYOUT":
            return DisputePayoutSaga(
                saga_id=model.saga_id,
                dispute_id=model.dispute_id,
                booking_id=model.booking_id,
                current_state=model.current_state,
                retry_count=model.retry_count,
                last_error=model.last_error,
                version=model.version,
                companion_wallet_id=model.companion_wallet_id,
                commission_rate=model.commission_rate,
            )
        else:
            raise ValueError(f"Unknown saga type: {model.saga_type}")
