<#
.SYNOPSIS
Script to run End-to-End tests for Identity Service on Windows (PowerShell) using isolated Docker Compose
#>

$ErrorActionPreference = "Stop"

Write-Host "====================================="
Write-Host "   Identity Service E2E Test Suite   "
Write-Host "====================================="

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location -Path "$ScriptDir\.."

Write-Host "[1/3] Building Test Environment..."
docker compose -f docker-compose.test.yml build

Write-Host "`n[2/3] Starting Infrastructure & Services..."
docker compose -f docker-compose.test.yml up -d --wait

Write-Host "`n[3/3] Running E2E Tests..."
# We don't stop on error for this specific command so we can always run cleanup
$ErrorActionPreference = "Continue"
docker compose -f docker-compose.test.yml run --rm e2e-runner
$TestExitCode = $LASTEXITCODE
$ErrorActionPreference = "Stop"

Write-Host "`n[Cleanup] Tearing down environment..."
docker compose -f docker-compose.test.yml down -v

if ($TestExitCode -eq 0) {
    Write-Host "`n✅ E2E Test Suite Completed Successfully!" -ForegroundColor Green
} else {
    Write-Host "`n❌ E2E Test Suite Failed!" -ForegroundColor Red
    exit $TestExitCode
}
