#!/usr/bin/env bash
# gen_proto.sh — Generate gRPC stubs from contracts/
# Output: gen/proto/ (*.pb.go, *_grpc.pb.go)

set -e

scriptDir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
serviceRoot="$(cd "$scriptDir/.." && pwd)"
repoRoot="$(cd "$serviceRoot/../.." && pwd)"
contracts="$repoRoot/contracts"
googleapis="$repoRoot/third_party/googleapis"
module="github.com/rent-a-girlfriend/booking-service"

# Add Go bin directory to PATH
export PATH=$PATH:$(go env GOPATH)/bin:$HOME/go/bin

echo ""
echo "==> [gen_proto] Generate gRPC stubs"
echo "    contracts : $contracts"
echo "    output    : $serviceRoot/gen/proto/"
echo ""

mkdir -p "$serviceRoot/gen/proto"

bookingProtos=(
    "common/v1/enums.proto"
    "booking/v1/messages/request_booking_request.proto"
    "booking/v1/messages/request_booking_response.proto"
    "booking/v1/messages/accept_booking_request.proto"
    "booking/v1/messages/accept_booking_response.proto"
    "booking/v1/messages/cancel_booking_request.proto"
    "booking/v1/messages/cancel_booking_response.proto"
    "booking/v1/messages/complete_booking_request.proto"
    "booking/v1/messages/complete_booking_response.proto"
    "booking/v1/messages/reject_booking_request.proto"
    "booking/v1/messages/reject_booking_response.proto"
    "booking/v1/messages/list_bookings_request.proto"
    "booking/v1/messages/list_bookings_response.proto"
    "booking/v1/messages/get_booking_request.proto"
    "booking/v1/messages/booking_detail_response.proto"
    "booking/v1/events/booking_accepted.proto"
    "booking/v1/events/booking_cancelled_early.proto"
    "booking/v1/events/booking_cancelled_late.proto"
    "booking/v1/events/booking_completed.proto"
    "booking/v1/events/booking_rejected.proto"
    "booking/v1/events/booking_requested.proto"
    "booking/v1/events/booking_timed_out.proto"
    "booking/v1/service/booking_service.proto"
)

financeProtos=(
    "finance/v1/service/finance_service.proto"
    "finance/v1/messages/finance_command_response.proto"
    "finance/v1/messages/freeze_coin_request.proto"
    "finance/v1/messages/get_wallet_request.proto"
    "finance/v1/messages/get_wallet_response.proto"
    "finance/v1/messages/process_payout_request.proto"
    "finance/v1/messages/refund_escrow_request.proto"
    "finance/v1/messages/transfer_to_escrow_request.proto"
    "finance/v1/events/coins_frozen.proto"
    "finance/v1/events/coins_unfrozen.proto"
    "finance/v1/events/escrow_created.proto"
    "finance/v1/events/escrow_failed.proto"
    "finance/v1/events/escrow_refunded.proto"
    "finance/v1/events/refund_failed.proto"
    "finance/v1/events/payout_processed.proto"
    "finance/v1/events/wallet_topped_up.proto"
)

profileProtos=(
    "profile/v1/service/profile_service.proto"
    "profile/v1/messages/create_profile_request.proto"
    "profile/v1/messages/update_profile_request.proto"
    "profile/v1/messages/approve_profile_request.proto"
    "profile/v1/messages/reject_profile_request.proto"
    "profile/v1/messages/profile_command_response.proto"
    "profile/v1/messages/create_scenario_request.proto"
    "profile/v1/messages/update_scenario_request.proto"
    "profile/v1/messages/delete_scenario_request.proto"
    "profile/v1/messages/scenario_command_response.proto"
    "profile/v1/messages/get_scenario_snapshot_request.proto"
    "profile/v1/messages/scenario_snapshot_response.proto"
    "profile/v1/messages/register_voice_intro_request.proto"
    "profile/v1/messages/register_album_image_request.proto"
    "profile/v1/messages/media_command_response.proto"
)

interactionProtos=(
    "interaction/v1/events/chat_room_created.proto"
    "interaction/v1/events/chat_room_creation_failed.proto"
)

disputeProtos=(
    "dispute/v1/events/dispute_created.proto"
    "dispute/v1/events/dispute_resolved.proto"
)

protoFiles=("${bookingProtos[@]}" "${financeProtos[@]}" "${profileProtos[@]}" "${interactionProtos[@]}" "${disputeProtos[@]}")

fullPaths=()
for f in "${protoFiles[@]}"; do
    fullPaths+=("$contracts/$f")
done

go_opts=(--go_opt=module="$module" --go_opt="Mcommon/v1/enums.proto=github.com/rent-a-girlfriend/booking-service/gen/proto;bookingv1")
go_grpc_opts=(--go-grpc_opt=module="$module" --go-grpc_opt="Mcommon/v1/enums.proto=github.com/rent-a-girlfriend/booking-service/gen/proto;bookingv1")

for f in "${financeProtos[@]}"; do
    go_opts+=(--go_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/financev1;financev1")
    go_grpc_opts+=(--go-grpc_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/financev1;financev1")
done

for f in "${profileProtos[@]}"; do
    go_opts+=(--go_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/profilev1;profilev1")
    go_grpc_opts+=(--go-grpc_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/profilev1;profilev1")
done

for f in "${interactionProtos[@]}"; do
    go_opts+=(--go_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/interactionv1;interactionv1")
    go_grpc_opts+=(--go-grpc_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/interactionv1;interactionv1")
done

for f in "${disputeProtos[@]}"; do
    go_opts+=(--go_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/disputev1;disputev1")
    go_grpc_opts+=(--go-grpc_opt="M$f=github.com/rent-a-girlfriend/booking-service/gen/proto/disputev1;disputev1")
done

# 1. Sinh gRPC code
protoc \
    -I "$contracts" \
    -I "$googleapis" \
    --go_out="$serviceRoot" \
    "${go_opts[@]}" \
    --go-grpc_out="$serviceRoot" \
    "${go_grpc_opts[@]}" \
    "${fullPaths[@]}"

# 2. Sinh HTTP Gateway code
protoc \
    -I "$contracts" \
    -I "$googleapis" \
    --grpc-gateway_out="$serviceRoot" \
    --grpc-gateway_opt=module="$module" \
    --grpc-gateway_opt="Mcommon/v1/enums.proto=github.com/rent-a-girlfriend/booking-service/gen/proto;bookingv1" \
    --grpc-gateway_opt=logtostderr=true \
    "$contracts/booking/v1/service/booking_service.proto"

echo "==> [gen_proto] Done! All files generated successfully inside gen/proto/"
