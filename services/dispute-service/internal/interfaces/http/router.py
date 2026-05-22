from typing import Optional, List
from fastapi import APIRouter, Depends, Header, HTTPException, Query, status
from pydantic import BaseModel
from internal.application.query import DisputeQueryService
from internal.domain.errors import DomainError

router = APIRouter(prefix="/api/v1")

from internal.bootstrap import get_query_service  # noqa: E402


class AuthInfo(BaseModel):
    user_id: Optional[str] = None
    user_role: Optional[str] = None
    user_email: Optional[str] = None


def get_admin_auth_info(
    user_id: Optional[str] = Header(None, alias="user-id"),
    user_role: Optional[str] = Header(None, alias="user-role"),
    user_email: Optional[str] = Header(None, alias="user-email"),
) -> AuthInfo:
    if not user_id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User authentication missing",
        )
    if user_role != "ADMIN":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Admin permission required",
        )
    return AuthInfo(
        user_id=user_id,
        user_role=user_role,
        user_email=user_email,
    )


# --- Response DTOs ---


class EvidenceDTO(BaseModel):
    evidence_id: str
    evidence_type: str
    content: str


class DisputeDTO(BaseModel):
    dispute_id: str
    booking_id: str
    reporter_id: str
    accused_id: str
    reason: str
    status: str
    admin_id: Optional[str] = None
    resolution: Optional[str] = None
    notes: Optional[str] = None
    version: int
    evidences: List[EvidenceDTO]

    @classmethod
    def from_domain(cls, domain) -> "DisputeDTO":
        return cls(
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
            evidences=[
                EvidenceDTO(
                    evidence_id=ev.evidence_id,
                    evidence_type=ev.evidence_type,
                    content=ev.content,
                )
                for ev in domain.evidences
            ],
        )


class SagaStateDTO(BaseModel):
    saga_id: str
    dispute_id: str
    booking_id: str
    saga_type: str
    current_state: str
    retry_count: int
    last_error: Optional[str] = None
    version: int

    @classmethod
    def from_domain(cls, domain) -> "SagaStateDTO":
        from internal.domain.aggregate import DisputeRefundSaga
        saga_type = "REFUND" if isinstance(domain, DisputeRefundSaga) else "PAYOUT"
        return cls(
            saga_id=domain.saga_id,
            dispute_id=domain.dispute_id,
            booking_id=domain.booking_id,
            saga_type=saga_type,
            current_state=domain.current_state,
            retry_count=domain.retry_count,
            last_error=domain.last_error,
            version=domain.version,
        )


# --- Routes ---


@router.get("/health", tags=["System"])
async def health_check():
    return {"status": "HEALTHY"}


@router.get("/disputes", response_model=dict, tags=["Admin Dispute Management"])
async def list_disputes(
    status_filter: Optional[str] = Query(None, alias="status"),
    page: int = Query(1, ge=1),
    page_size: int = Query(10, ge=1, le=50),
    auth_info: AuthInfo = Depends(get_admin_auth_info),
    query_service: DisputeQueryService = Depends(get_query_service),
):
    try:
        offset = (page - 1) * page_size
        disputes, total = await query_service.list_disputes(
            status=status_filter, offset=offset, limit=page_size
        )
        return {
            "disputes": [DisputeDTO.from_domain(d) for d in disputes],
            "total": total,
            "page": page,
            "page_size": page_size,
        }
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to query disputes: {str(e)}",
        )


@router.get("/disputes/{dispute_id}", response_model=DisputeDTO, tags=["Admin Dispute Management"])
async def get_dispute_detail(
    dispute_id: str,
    auth_info: AuthInfo = Depends(get_admin_auth_info),
    query_service: DisputeQueryService = Depends(get_query_service),
):
    try:
        dispute = await query_service.get_dispute_detail(dispute_id)
        return DisputeDTO.from_domain(dispute)
    except DomainError as e:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=str(e))
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e)
        )


@router.get("/disputes/{dispute_id}/saga", response_model=Optional[SagaStateDTO], tags=["Admin Dispute Management"])
async def get_saga_state(
    dispute_id: str,
    auth_info: AuthInfo = Depends(get_admin_auth_info),
    query_service: DisputeQueryService = Depends(get_query_service),
):
    try:
        saga = await query_service.get_saga_state(dispute_id)
        if not saga:
            return None
        return SagaStateDTO.from_domain(saga)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e)
        )
