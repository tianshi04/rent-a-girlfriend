from fastapi import Request
from fastapi.exceptions import RequestValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException
from fastapi.responses import JSONResponse
from internal.domain.errors import DomainError


def map_http_status_to_grpc_code(status_code: int) -> int:
    """
    Maps HTTP status codes to gRPC status codes according to google.rpc.Code
    """
    mapping = {
        400: 3,  # INVALID_ARGUMENT
        401: 16,  # UNAUTHENTICATED
        403: 7,  # PERMISSION_DENIED
        404: 5,  # NOT_FOUND
        409: 6,  # ALREADY_EXISTS
        422: 3,  # INVALID_ARGUMENT
        500: 13,  # INTERNAL
        501: 12,  # UNIMPLEMENTED
        503: 14,  # UNAVAILABLE
    }
    return mapping.get(status_code, 2)  # 2 is UNKNOWN


async def domain_error_handler(request: Request, exc: DomainError) -> JSONResponse:
    # Most domain errors are considered bad requests (validation/business rule violations)
    status_code = 400
    msg = str(exc)
    if "not found" in msg.lower():
        status_code = 404

    grpc_code = map_http_status_to_grpc_code(status_code)

    return JSONResponse(
        status_code=status_code,
        content={"code": grpc_code, "message": msg, "details": []},
    )


async def http_exception_handler(
    request: Request, exc: StarletteHTTPException
) -> JSONResponse:
    grpc_code = map_http_status_to_grpc_code(exc.status_code)
    return JSONResponse(
        status_code=exc.status_code,
        content={"code": grpc_code, "message": str(exc.detail), "details": []},
    )


async def validation_exception_handler(
    request: Request, exc: RequestValidationError
) -> JSONResponse:
    grpc_code = map_http_status_to_grpc_code(422)
    details = []
    for error in exc.errors():
        field = ".".join(str(loc) for loc in error.get("loc", []))
        details.append({"field": field, "issue": error.get("msg")})

    return JSONResponse(
        status_code=422,
        content={"code": grpc_code, "message": "Validation Error", "details": details},
    )
