<#
.SYNOPSIS
Script to run End-to-End tests for Identity Service on Windows (PowerShell)
#>

$ErrorActionPreference = "Stop"

Write-Host "====================================="
Write-Host "   Identity Service E2E Test Suite   "
Write-Host "====================================="

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location -Path "$ScriptDir\.."

# Export necessary environment variables for testing
$env:APP_ENV="test"
$env:ENABLE_TEST_ROUTES="true"
$env:GIN_MODE="release"
$env:DB_HOST="localhost"
$env:DB_PORT="5432"
$env:DB_USER="postgres"
$env:DB_PASSWORD="postgres"
$env:DB_NAME="identity_db"
$env:DB_SSLMODE="disable"
$env:DATABASE_URL="postgres://${env:DB_USER}:${env:DB_PASSWORD}@${env:DB_HOST}:${env:DB_PORT}/${env:DB_NAME}?sslmode=${env:DB_SSLMODE}"
$env:REDIS_URL="redis://localhost:6379"
$env:KAFKA_BROKERS="localhost:9092,localhost:9093,localhost:9094"
$env:OUTBOX_POLLING_INTERVAL_MS="500"
$env:SERVER_PORT="8081"
$env:GRPC_PORT="50051"

Write-Host "[1/4] Starting dependencies via docker-compose..."
# Assuming user has docker desktop installed
# docker-compose -f ../../infrastructure/kafka/docker-compose.yml up -d
# docker-compose up -d db redis

# (Skipping automatic wait for now as it requires specific netcmd or powershell loops)
Write-Host "Please ensure Postgres(5432), Redis(6379), and Kafka(9092) are running."
Start-Sleep -Seconds 2

Write-Host "[2/4] Starting Identity Service in background..."
$ServerProcess = Start-Process -FilePath "go" -ArgumentList "run", "cmd/server/main.go" -NoNewWindow -PassThru

Write-Host "Waiting for Identity Service HTTP to be ready..."
$isUp = $false
for ($i = 0; $i -lt 30; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8081/health" -Method Head -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            $isUp = $true
            break
        }
    } catch {}
    Write-Host -NoNewline "."
    Start-Sleep -Seconds 1
}

if (-not $isUp) {
    Write-Host "`nFailed to start Identity Service."
    Stop-Process -Id $ServerProcess.Id -Force
    exit 1
}

Write-Host "`nService is up!"

Write-Host "[3/4] Running E2E Tests..."
go test -v ./tests/e2e/...

$TestExitCode = $LASTEXITCODE

Write-Host "[4/4] Tearing down environment..."
Stop-Process -Id $ServerProcess.Id -Force
Write-Host "Identity Service stopped."

if ($TestExitCode -eq 0) {
    Write-Host "E2E Test Suite Completed Successfully!" -ForegroundColor Green
} else {
    Write-Host "E2E Test Suite Failed!" -ForegroundColor Red
    exit $TestExitCode
}
