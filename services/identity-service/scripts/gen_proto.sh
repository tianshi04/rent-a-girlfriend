#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SERVICE_ROOT/../.." && pwd)"
CONTRACTS="$REPO_ROOT/contracts"
GOOGLEAPIS="$REPO_ROOT/third_party/googleapis"
MODULE="github.com/rent-a-girlfriend/identity-service"


# Automatically add Go bin directory to PATH so protoc can find installed plugins
GO_PATH="$(go env GOPATH)"
if [ -n "$GO_PATH" ]; then
    export PATH="$PATH:$GO_PATH/bin"
fi
export PATH="$PATH:$HOME/go/bin"

echo ""
echo "==> [gen_proto] Generate gRPC & HTTP Gateway stubs"
echo "    contracts : $CONTRACTS"
echo "    output    : $SERVICE_ROOT/internal/gen/proto/"
echo ""

PROTO_FILES=(
    "common/common.proto"
    "identity/v1/messages/requests/init_google_auth_request.proto"
    "identity/v1/messages/requests/login_google_request.proto"
    "identity/v1/messages/requests/refresh_token_request.proto"
    "identity/v1/messages/requests/logout_request.proto"
    "identity/v1/messages/requests/get_account_request.proto"
    "identity/v1/messages/requests/list_upgrade_requests_request.proto"
    "identity/v1/messages/requests/request_upgrade_request.proto"
    "identity/v1/messages/requests/approve_upgrade_request.proto"
    "identity/v1/messages/requests/reject_upgrade_request.proto"
    "identity/v1/messages/requests/lock_account_request.proto"
    "identity/v1/messages/requests/unlock_account_request.proto"
    "identity/v1/messages/responses/init_google_auth_response.proto"
    "identity/v1/messages/responses/token_response.proto"
    "identity/v1/messages/responses/account_response.proto"
    "identity/v1/messages/responses/account_role.proto"
    "identity/v1/messages/responses/account_status.proto"
    "identity/v1/messages/responses/upgrade_status.proto"
    "identity/v1/messages/responses/upgrade_request_item.proto"
    "identity/v1/messages/responses/list_upgrade_requests_response.proto"
    "identity/v1/events/account_locked_event.proto"
    "identity/v1/events/violation_recorded_event.proto"
    "identity/v1/events/upgrade_requested_event.proto"
    "identity/v1/events/upgrade_approved_event.proto"
    "identity/v1/events/upgrade_rejected_event.proto"
    "identity/v1/service/identity_service.proto"
)

FULL_PATHS=()
for file in "${PROTO_FILES[@]}"; do
    FULL_PATHS+=("$CONTRACTS/$file")
done

# 1. Sinh gRPC code
protoc \
    -I "$CONTRACTS" \
    -I "$GOOGLEAPIS" \
    --go_out="$SERVICE_ROOT" \
    --go_opt=module="$MODULE" \
    --go-grpc_out="$SERVICE_ROOT" \
    --go-grpc_opt=module="$MODULE" \
    "${FULL_PATHS[@]}"

# 2. Sinh HTTP Gateway code
protoc \
    -I "$CONTRACTS" \
    -I "$GOOGLEAPIS" \
    --grpc-gateway_out="$SERVICE_ROOT" \
    --grpc-gateway_opt=module="$MODULE" \
    --grpc-gateway_opt=logtostderr=true \
    "$CONTRACTS/identity/v1/service/identity_service.proto"

echo "==> [gen_proto] Done! All files generated successfully inside internal/gen/proto/"
