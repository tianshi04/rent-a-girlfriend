# Notification Service Smoke Test Orchestrator
$ErrorActionPreference = "Stop"

# Get script's parent folder
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "[*] STARTING NOTIFICATION SERVICE SMOKE TEST" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Build the Notification Service Docker Image
Write-Host "`n[+] Step 1: Building Docker image..." -ForegroundColor Yellow
docker build -t rentagf/notification-service:smoke -f ../../Dockerfile ../../

# 2. Run Docker Compose
Write-Host "`n[+] Step 2: Spinning up container environment..." -ForegroundColor Yellow
docker compose -f docker-compose.smoke.yml up -d

# 3. Poll Actuator Health Endpoint
Write-Host "`n[*] Step 3: Waiting for Actuator health status (UP)..." -ForegroundColor Yellow
$url = "http://localhost:8084/actuator/health"
$maxRetries = 30
$retryCount = 0
$started = $false

while ($retryCount -lt $maxRetries) {
    $response = $null
    $errorMsg = $null
    try {
        $response = Invoke-RestMethod -Uri $url -Method Get -TimeoutSec 2
    } catch {
        $errorMsg = $_.Exception.Message
    }

    if ($response -and $response.status -eq "UP") {
        Write-Host "`n[V] SUCCESS: Notification Service started successfully!" -ForegroundColor Green
        Write-Host "Actuator Response: " -NoNewline
        Write-Host (ConvertTo-Json $response -Depth 3) -ForegroundColor Gray
        $started = $true
        break
    } else {
        Write-Host "." -NoNewline
    }
    $retryCount++
    Start-Sleep -Seconds 2
}

if (-not $started) {
    Write-Host "`n[X] FAILED: Service did not start successfully or report UP status within timeout." -ForegroundColor Red
    Write-Host "Checking service container logs:" -ForegroundColor Yellow
    docker logs notification-service-smoke
    
    Write-Host "`n[-] Cleaning up environment..." -ForegroundColor Yellow
    docker compose -f docker-compose.smoke.yml down
    exit 1
}

# 4. Cleanup
Write-Host "`n[-] Step 4: Cleaning up environment..." -ForegroundColor Yellow
docker compose -f docker-compose.smoke.yml down
Write-Host "Smoke test completed and cleaned up." -ForegroundColor Green
