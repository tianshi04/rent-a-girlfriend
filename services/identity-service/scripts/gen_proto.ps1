#!/usr/bin/env pwsh
# gen_proto.ps1 — Generate cả gRPC và HTTP Gateway code từ contracts/
# Output: internal/gen/proto/ (*.pb.go, *_grpc.pb.go, *.pb.gw.go)

$ErrorActionPreference = "Stop"

$scriptDir   = $PSScriptRoot
$serviceRoot = Resolve-Path "$scriptDir\.."
$repoRoot    = Resolve-Path "$serviceRoot\..\.."
$contracts   = "$repoRoot\contracts"
$googleapis  = "$repoRoot\third_party\googleapis"
$module      = "github.com/rent-a-girlfriend/identity-service"

# Automatically add Go bin directory to PATH so protoc can find installed plugins
$goGopath = go env GOPATH
if ($goGopath) {
    $Env:Path += ";$goGopath\bin"
}
$Env:Path += ";$env:USERPROFILE\go\bin"

Write-Host ""
Write-Host "==> [gen_proto] Generate gRPC & HTTP Gateway stubs"
Write-Host "    contracts : $contracts"
Write-Host "    output    : $serviceRoot\gen\proto\"
Write-Host ""

$protoFiles = @(
    "common\common.proto",
    "identity\v1\messages\requests\init_google_auth_request.proto",
    "identity\v1\messages\requests\login_google_request.proto",
    "identity\v1\messages\requests\refresh_token_request.proto",
    "identity\v1\messages\requests\logout_request.proto",
    "identity\v1\messages\requests\get_account_request.proto",
    "identity\v1\messages\requests\list_upgrade_requests_request.proto",
    "identity\v1\messages\requests\request_upgrade_request.proto",
    "identity\v1\messages\requests\approve_upgrade_request.proto",
    "identity\v1\messages\requests\reject_upgrade_request.proto",
    "identity\v1\messages\requests\lock_account_request.proto",
    "identity\v1\messages\requests\unlock_account_request.proto",
    "identity\v1\messages\responses\init_google_auth_response.proto",
    "identity\v1\messages\responses\token_response.proto",
    "identity\v1\messages\responses\account_response.proto",
    "identity\v1\messages\responses\account_role.proto",
    "identity\v1\messages\responses\account_status.proto",
    "identity\v1\messages\responses\upgrade_status.proto",
    "identity\v1\messages\responses\upgrade_request_item.proto",
    "identity\v1\messages\responses\list_upgrade_requests_response.proto",
    "identity\v1\events\account_locked_event.proto",
    "identity\v1\events\violation_recorded_event.proto",
    "identity\v1\events\upgrade_requested_event.proto",
    "identity\v1\events\upgrade_approved_event.proto",
    "identity\v1\events\upgrade_rejected_event.proto",
    "identity\v1\service\identity_service.proto"
)

$fullPaths = $protoFiles | ForEach-Object { "$contracts\$_" }

# 1. Sinh gRPC code
& protoc `
    -I $contracts `
    -I $googleapis `
    --go_out=$serviceRoot `
    --go_opt=module=$module `
    --go-grpc_out=$serviceRoot `
    --go-grpc_opt=module=$module `
    @fullPaths

if ($LASTEXITCODE -ne 0) {
    Write-Error "protoc (gRPC) failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}

# 2. Sinh HTTP Gateway code
& protoc `
    -I $contracts `
    -I $googleapis `
    --grpc-gateway_out=$serviceRoot `
    --grpc-gateway_opt=module=$module `
    --grpc-gateway_opt=logtostderr=true `
    "$contracts\identity\v1\service\identity_service.proto"

if ($LASTEXITCODE -ne 0) {
    Write-Error "protoc (HTTP Gateway) failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}

Write-Host "==> [gen_proto] Done! All files generated successfully inside internal/gen/proto/"
