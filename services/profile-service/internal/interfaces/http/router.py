from typing import Optional
from fastapi import APIRouter, Depends, Header, HTTPException, Query, status
from pydantic import BaseModel, Field, model_validator
from internal.application.command import MediaCommandService
from internal.application.query import ProfileQueryService
from internal.domain.errors import DomainError

router = APIRouter(prefix="/api/v1")

from internal.bootstrap import get_query_service, get_media_cmd  # noqa: E402

# --- Input/Output Schemas ---


class PresignedUrlRequest(BaseModel):
    assetType: str = Field(
        ..., description="IMAGE or VOICE", json_schema_extra={"example": "VOICE"}
    )
    sizeBytes: int = Field(
        ..., description="File size in bytes", json_schema_extra={"example": 1048576}
    )
    durationSeconds: Optional[int] = Field(
        None, description="Required for VOICE type", json_schema_extra={"example": 25}
    )
    contentType: str = Field(
        ..., description="MIME content type", json_schema_extra={"example": "audio/mp3"}
    )

    @model_validator(mode="after")
    def validate_business_logic(self) -> "PresignedUrlRequest":
        if self.assetType not in ("IMAGE", "VOICE"):
            raise ValueError("assetType must be either IMAGE or VOICE")

        if self.assetType == "VOICE":
            if self.durationSeconds is None:
                raise ValueError("durationSeconds is required when assetType is VOICE")
            if self.durationSeconds <= 0:
                raise ValueError("durationSeconds must be greater than 0")
            if self.durationSeconds > 30:
                raise ValueError("durationSeconds must not exceed 30 seconds")
            if self.sizeBytes > 5 * 1024 * 1024:
                raise ValueError("VOICE size must not exceed 5MB")
        elif self.assetType == "IMAGE":
            if self.sizeBytes > 2 * 1024 * 1024:
                raise ValueError("IMAGE size must not exceed 2MB")

        return self


class PresignedUrlResponse(BaseModel):
    uploadUrl: str
    fileUrl: str


class AuthInfo(BaseModel):
    user_id: Optional[str] = None
    user_role: Optional[str] = None
    user_status: Optional[str] = None
    user_email: Optional[str] = None


def get_auth_info(
    user_id: Optional[str] = Header(None, alias="user-id"),
    user_role: Optional[str] = Header(None, alias="user-role"),
    user_status: Optional[str] = Header(None, alias="user-status"),
    user_email: Optional[str] = Header(None, alias="user-email"),
) -> AuthInfo:
    if not user_id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User authentication missing",
        )
    return AuthInfo(
        user_id=user_id,
        user_role=user_role,
        user_status=user_status,
        user_email=user_email,
    )


# --- Routes ---


@router.get("/profile/companions", tags=["Catalogue Query"])
async def search_companions(
    name: Optional[str] = Query(None),
    city: Optional[str] = Query(None),
    min_price: Optional[int] = Query(None),
    max_price: Optional[int] = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(10, ge=1, le=50),
    query_service: ProfileQueryService = Depends(get_query_service),
):
    try:
        results = await query_service.search_companions(
            name=name,
            city=city,
            min_price=min_price,
            max_price=max_price,
            page=page,
            page_size=page_size,
        )
        return results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to query catalogue: {str(e)}",
        )


@router.get("/profile/companions/{companion_id}", tags=["Catalogue Query"])
async def get_companion_detail(
    companion_id: str, query_service: ProfileQueryService = Depends(get_query_service)
):
    try:
        detail = await query_service.get_companion_detail(companion_id, public=True)
        return detail
    except DomainError as e:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=str(e))
    except Exception as e:
        if isinstance(e, HTTPException):
            raise e
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e)
        )


@router.get("/profile/me", tags=["Profile Management Query"])
async def get_my_profile(
    auth_info: AuthInfo = Depends(get_auth_info),
    query_service: ProfileQueryService = Depends(get_query_service),
):
    try:
        detail = await query_service.get_my_profile(auth_info.user_id)
        return detail
    except DomainError as e:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=str(e))
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e)
        )


@router.post(
    "/profile/me/media/presigned-urls",
    response_model=PresignedUrlResponse,
    tags=["Media Management"],
)
async def request_presigned_url(
    payload: PresignedUrlRequest,
    auth_info: AuthInfo = Depends(get_auth_info),
    media_cmd: MediaCommandService = Depends(get_media_cmd),
):
    try:
        presign_data = await media_cmd.request_presigned_url(
            companion_id=auth_info.user_id,
            asset_type=payload.assetType,
            size_bytes=payload.sizeBytes,
            duration_seconds=payload.durationSeconds,
            content_type=payload.contentType,
        )
        return PresignedUrlResponse(
            uploadUrl=presign_data["uploadUrl"], fileUrl=presign_data["fileUrl"]
        )
    except DomainError as e:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(e))
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e)
        )
